package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/gemini"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// ScanReceiptPageData is the view model for templates/scan_receipt.html.
type ScanReceiptPageData struct {
	PageData

	// Result is the last scan's parsed data (see ReceiptScanResult). Nil
	// until a photo has been submitted.
	Result *ReceiptScanResult
	Error  string

	// ActiveTab is "scan" (default) or "queue" — which panel the tab
	// switcher shows first on page load (before any client-side JS takes
	// over). See templates/scripts.html's "tabs-script".
	ActiveTab string

	// QueueItems is every pending_pantry_items row matching SelectedStatuses,
	// rejected items first (see BuildQueueItems).
	QueueItems []QueueItemView
	// SelectedStatuses backs the Queue tab's status checkboxes' checked state.
	SelectedStatuses map[string]bool
	// StatusOptions backs the Queue tab's status checkbox list.
	StatusOptions []QueueStatusOption
}

// ReceiptScanResult is the standard JSON shape Gemini is asked to return for
// a receipt scan (enforced via receiptSchema, not just prompt wording): one
// entry per purchased item — can/bottle deposit lines excluded — plus the
// receipt's total and any taxes.
type ReceiptScanResult struct {
	Items []ReceiptItem `json:"items"`
	Total *float64      `json:"total"`
	Taxes []ReceiptTax  `json:"taxes"`
}

// ReceiptItem is one purchased line item. Quantity/Price/Code are nil when
// the receipt doesn't show them for that item.
type ReceiptItem struct {
	Name     string   `json:"name"`
	Quantity *float64 `json:"quantity"`
	// Unit is the quantity's unit of measurement (e.g. "kg", "lb", "L") as
	// printed or implied on the receipt — not matched against our own units
	// table. Defaulted to "unit" after decoding if Gemini didn't return one.
	Unit  *string  `json:"unit"`
	Price *float64 `json:"price"`
	// Code is whatever product code/SKU is printed on the receipt itself —
	// not matched against our own ingredients table.
	Code *string `json:"code"`
	// UPC is the full 12-digit UPC-A code derived from Code (see upcFromCode),
	// nil if Code doesn't look like a bare UPC without its check digit. Not
	// part of receiptSchema — computed after Gemini's response is decoded.
	UPC *string `json:"-"`
}

// ReceiptTax is one named tax line (e.g. "GST", "QST") and its amount.
type ReceiptTax struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// receiptSchema is the JSON standard ReceiptScanResult decodes — its field
// names must match ReceiptScanResult/ReceiptItem/ReceiptTax's `json` tags
// exactly, since Gemini's structured output uses these as the literal
// response keys.
var receiptSchema = &gemini.Schema{
	Type: "OBJECT",
	Properties: map[string]*gemini.Schema{
		"items": {
			Type: "ARRAY",
			Items: &gemini.Schema{
				Type: "OBJECT",
				Properties: map[string]*gemini.Schema{
					"name":     {Type: "STRING"},
					"quantity": {Type: "NUMBER", Nullable: true},
					"unit":     {Type: "STRING", Nullable: true},
					"price":    {Type: "NUMBER", Nullable: true},
					"code":     {Type: "STRING", Nullable: true},
				},
				Required: []string{"name"},
			},
		},
		"total": {Type: "NUMBER", Nullable: true},
		"taxes": {
			Type: "ARRAY",
			Items: &gemini.Schema{
				Type: "OBJECT",
				Properties: map[string]*gemini.Schema{
					"name":   {Type: "STRING"},
					"amount": {Type: "NUMBER"},
				},
				Required: []string{"name", "amount"},
			},
		},
	},
	Required: []string{"items", "taxes"},
}

// maxReceiptPhotoBytes caps the upload generously for a phone photo while
// keeping the request small enough to send to Gemini inline (no Files API).
const maxReceiptPhotoBytes = 10 << 20 // 10 MiB

// defaultReceiptUnit is used for an item's Unit when Gemini doesn't return
// one — e.g. the receipt just counts individual items rather than weighing
// or measuring them.
const defaultReceiptUnit = "unit"

// defaultReceiptQuantity is used for an item's Quantity when Gemini doesn't
// return one — pending_pantry_items.quantity is NOT NULL, and a receipt line
// with no quantity shown is a single unit of that item.
const defaultReceiptQuantity float64 = 1

// pending_pantry_items.status values. An item with a derivable UPC waits for
// review ("pending"); one without is rejected immediately since there's no
// barcode to ever match it against. "processing"/"approved" are set by a
// review flow that doesn't exist yet — scanned items never reach them on
// their own.
const (
	pendingPantryStatusPending    = "pending"
	pendingPantryStatusProcessing = "processing"
	pendingPantryStatusApproved   = "approved"
	pendingPantryStatusRejected   = "rejected"
)

// upcDigits is how many digits a UPC-A code has before its trailing check
// digit — receipts print the bare 11 digits (or fewer, missing leading
// zeros) but never the check digit itself.
const upcDigits = 11

// upcFromCode derives a full 12-digit UPC-A code from a receipt's printed
// product code: left-pads it with zeros to 11 digits, then appends the
// computed check digit. Returns false if code isn't purely numeric or
// already longer than 11 digits, since it then doesn't look like a bare UPC
// missing its check digit.
func upcFromCode(code string) (string, bool) {
	if code == "" || len(code) > upcDigits {
		return "", false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	padded := strings.Repeat("0", upcDigits-len(code)) + code
	return padded + string(rune('0'+upcCheckDigit(padded))), true
}

// upcCheckDigit computes the UPC-A check digit for an 11-digit numeric
// string: digits at odd positions (1st, 3rd, ...) are weighted 3x, digits
// at even positions are weighted 1x, and the check digit is whatever brings
// that sum up to the next multiple of 10.
func upcCheckDigit(elevenDigits string) int {
	sum := 0
	for i, r := range elevenDigits {
		d := int(r - '0')
		if i%2 == 0 {
			d *= 3
		}
		sum += d
	}
	return (10 - sum%10) % 10
}

// receiptPrompt asks Gemini to read a photographed grocery receipt and
// extract what was bought as JSON matching receiptSchema. Can/bottle deposit
// lines are explicitly excluded (they aren't a purchased item); the schema
// only constrains shape/types, not this kind of semantic judgment call, so
// it still needs spelling out here.
const receiptPrompt = `This image is a photo of a grocery store receipt. ` +
	`Extract every item that was purchased: its name, the quantity bought ` +
	`(if shown), the unit that quantity is measured in (e.g. "kg", "g", ` +
	`"lb", "L", "mL" — if the receipt just counts individual items, use ` +
	`"unit"), the price paid (if shown), and any product code/SKU ` +
	`printed next to it on the receipt (if shown). ` +
	`Ignore can/bottle deposit lines entirely (e.g. "consigne", "deposit", ` +
	`"CRV") — do not list them as a purchased item. ` +
	`Also extract the receipt's total price, and any taxes shown (e.g. ` +
	`GST/QST/TPS/TVQ/VAT) as name/amount pairs. ` +
	`If the image isn't a receipt or nothing on it is legible, return an ` +
	`empty items list rather than guessing.`

// ScanReceiptPage renders GET /scan-receipt: the "take a photo" form.
func (h *Handler) ScanReceiptPage(w http.ResponseWriter, r *http.Request) {
	h.renderScanReceiptPage(w, r, ScanReceiptPageData{
		PageData: h.newPageData(r, "Scan receipt", ""),
	})
}

// ScanReceiptSubmit handles POST /scan-receipt: reads the uploaded photo,
// asks Gemini to extract what was bought (as JSON matching receiptSchema),
// and renders the parsed result directly.
//
// Unlike every other form in this app, this does not redirect on success —
// there is nothing persisted to reload from a redirect, and round-tripping
// a photo through a redirect URL isn't practical, so the result page is
// rendered straight from the POST.
func (h *Handler) ScanReceiptSubmit(w http.ResponseWriter, r *http.Request) {
	data := ScanReceiptPageData{PageData: h.newPageData(r, "Scan receipt", "")}

	r.Body = http.MaxBytesReader(w, r.Body, maxReceiptPhotoBytes)
	if err := r.ParseMultipartForm(maxReceiptPhotoBytes); err != nil {
		data.Error = "photo is too large or the form was invalid"
		h.renderScanReceiptPage(w, r, data)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		data.Error = "please choose or take a photo"
		h.renderScanReceiptPage(w, r, data)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("read receipt photo", "error", err)
		data.Error = "failed to read the uploaded photo"
		h.renderScanReceiptPage(w, r, data)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	raw, err := h.gemini.GenerateFromImage(r.Context(), receiptPrompt, imageBytes, mimeType, receiptSchema)
	if err != nil {
		h.logger.Error("gemini receipt scan", "error", err)
		data.Error = "failed to read the receipt, please try again"
		h.renderScanReceiptPage(w, r, data)
		return
	}

	var result ReceiptScanResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		h.logger.Error("decode gemini receipt response", "error", err, "raw", raw)
		data.Error = "got an unreadable response back, please try again"
		h.renderScanReceiptPage(w, r, data)
		return
	}

	for i, item := range result.Items {
		if item.Unit == nil || strings.TrimSpace(*item.Unit) == "" {
			defaultUnit := defaultReceiptUnit
			result.Items[i].Unit = &defaultUnit
		}
		if item.Code == nil {
			continue
		}
		if upc, ok := upcFromCode(*item.Code); ok {
			result.Items[i].UPC = &upc
		}
	}

	// Every item is queued. One with no UPC (no code printed on the receipt,
	// or one that didn't look like a bare UPC) is saved with an empty upc
	// and status "rejected" straight away, since there's no barcode to ever
	// match it against.
	for _, item := range result.Items {
		upc := ""
		status := pendingPantryStatusRejected
		if item.UPC != nil {
			upc = *item.UPC
			status = pendingPantryStatusPending
		}
		quantity := defaultReceiptQuantity
		if item.Quantity != nil {
			quantity = *item.Quantity
		}
		if _, err := h.store.CreatePendingPantryItem(r.Context(), store.PendingPantryItemInput{
			UPC:      upc,
			Name:     item.Name,
			Quantity: quantity,
			Unit:     *item.Unit,
			Price:    item.Price,
			Status:   status,
		}); err != nil {
			h.logger.Error("save pending pantry item", "error", err, "upc", upc)
		}
	}

	data.Result = &result
	h.renderScanReceiptPage(w, r, data)
}

// renderScanReceiptPage fills in the Queue tab's fields (shared by both the
// plain GET page and a POST scan result) and renders scan_receipt.html.
func (h *Handler) renderScanReceiptPage(w http.ResponseWriter, r *http.Request, data ScanReceiptPageData) {
	if err := h.loadQueueTab(r, &data); err != nil {
		h.logger.Error("list pending pantry items", "error", err)
		http.Error(w, "failed to load queue", http.StatusInternalServerError)
		return
	}

	ts, ok := h.templatesCache["scan_receipt.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "scan_receipt.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// loadQueueTab reads the Queue tab's state from r's query string — which tab
// is active, and (via ?status_filter=1&status=...) whether a status filter
// was explicitly submitted — and populates data's ActiveTab/StatusOptions/
// SelectedStatuses/QueueItems accordingly.
//
// With no explicit filter (a fresh page load, or the "Queue" tab link),
// defaultQueueStatuses is used. An explicit filter with zero statuses
// checked (the user unchecked every box) is honored as-is — an empty
// result — rather than falling back to "no filter" like an unset query
// param would.
func (h *Handler) loadQueueTab(r *http.Request, data *ScanReceiptPageData) error {
	q := r.URL.Query()

	data.ActiveTab = "scan"
	if q.Get("tab") == "queue" {
		data.ActiveTab = "queue"
	}
	data.StatusOptions = queueStatusOptions

	statuses := defaultQueueStatuses
	explicitFilter := q.Has("status_filter")
	if explicitFilter {
		statuses = q["status"]
	}

	data.SelectedStatuses = make(map[string]bool, len(statuses))
	for _, s := range statuses {
		data.SelectedStatuses[s] = true
	}

	// An explicit filter selecting nothing means "show nothing" — skip the
	// query rather than letting ListPendingPantryItems treat an empty slice
	// as "no filter".
	if explicitFilter && len(statuses) == 0 {
		data.QueueItems = nil
		return nil
	}

	rows, err := h.store.ListPendingPantryItems(r.Context(), store.PendingPantryItemFilter{Statuses: statuses})
	if err != nil {
		return err
	}
	data.QueueItems = BuildQueueItems(rows)
	return nil
}
