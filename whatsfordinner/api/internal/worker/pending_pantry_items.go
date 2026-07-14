// Package worker runs background jobs for whatsfordinner. Currently just
// PendingPantryItemsWorker, which resolves scanned receipt items against
// OpenFoodFacts.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// openFoodFactsUserAgent identifies this app to OpenFoodFacts, per their
// policy: "AppName/Version (ContactEmail)", so a request isn't mistaken for
// a bot and blocked.
const openFoodFactsUserAgent = "whatsfordinner/1.0 (shamann4242@gmail.com)"

// openFoodFactsTimeout bounds each product lookup so a slow/hanging response
// can't stall an entire batch.
const openFoodFactsTimeout = 10 * time.Second

// PendingPantryItemsWorker periodically claims a batch of pending_pantry_items
// rows and resolves each one against OpenFoodFacts by UPC: found -> Approved,
// anything else -> Rejected, no retries.
type PendingPantryItemsWorker struct {
	store      *store.Store
	logger     *slog.Logger
	interval   time.Duration
	batchSize  int
	locale     string
	httpClient *http.Client
}

// New builds a worker. interval/batchSize/locale come from config.Config's
// QueueWorker*/OpenFoodFactsLocale fields, so the batch size, cadence and
// country catalog can be tuned without a code change.
func New(s *store.Store, logger *slog.Logger, interval time.Duration, batchSize int, locale string) *PendingPantryItemsWorker {
	return &PendingPantryItemsWorker{
		store:      s,
		logger:     logger,
		interval:   interval,
		batchSize:  batchSize,
		locale:     locale,
		httpClient: &http.Client{Timeout: openFoodFactsTimeout},
	}
}

// Run ticks every w.interval, processing one batch per tick, until ctx is
// canceled. Intended to be started in its own goroutine from main.
func (w *PendingPantryItemsWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch claims up to w.batchSize oldest-first pending items and
// resolves each one against OpenFoodFacts, one request at a time (see Run's
// doc comment on the rate-limit reasoning) — no retries: a lookup failure
// just rejects the item outright.
func (w *PendingPantryItemsWorker) processBatch(ctx context.Context) {
	items, err := w.store.ClaimPendingPantryItemsForProcessing(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("claim pending pantry items", "error", err)
		return
	}

	for _, item := range items {
		found, err := w.productFound(ctx, item.UPC)
		status := store.PendingPantryStatusApproved
		switch {
		case err != nil:
			status = store.PendingPantryStatusRejected
			w.logger.Warn("openfoodfacts product lookup failed", "upc", item.UPC, "error", err)
		case !found:
			status = store.PendingPantryStatusRejected
			w.logger.Debug("openfoodfacts product not found", "upc", item.UPC)
		}

		if err := w.store.SetPendingPantryItemStatus(ctx, item.ID, status); err != nil {
			w.logger.Error("update pending pantry item status", "id", item.ID, "error", err)
			continue
		}
		w.logger.Debug("resolved pending pantry item", "id", item.ID, "upc", item.UPC, "status", status)
	}
}

// productFound reports whether upc resolves to a real product on
// OpenFoodFacts' live (production) API — no Basic Auth needed there, unlike
// the staging/sandbox host.
//
// Only the envelope needed to answer that (HTTP status + presence of a
// non-null "product" object) is decoded. OpenFoodFacts' full product schema
// is enormous and its fields aren't consistently typed across products
// (e.g. some are a string on one product and a number on another) — trying
// to unmarshal all of it into a fixed Go struct is exactly what caused a
// third-party client library to misreport real products as lookup failures,
// so this deliberately only reads the couple of fields it actually needs.
func (w *PendingPantryItemsWorker) productFound(ctx context.Context, upc string) (bool, error) {
	endpoint := fmt.Sprintf(
		"https://%s.openfoodfacts.org/api/v3.6/product/%s.json",
		url.PathEscape(w.locale), url.PathEscape(upc),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", openFoodFactsUserAgent)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("openfoodfacts: unexpected status %s", resp.Status)
	}

	var body struct {
		Product json.RawMessage `json:"product"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, err
	}
	return len(body.Product) > 0 && string(body.Product) != "null", nil
}
