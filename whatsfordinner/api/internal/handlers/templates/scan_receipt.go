package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/gemini"
)

// ScanReceiptPageData is the view model for templates/scan_receipt.html.
type ScanReceiptPageData struct {
	PageData

	// Result is the last scan's parsed data (see ReceiptScanResult). Nil
	// until a photo has been submitted.
	Result *ReceiptScanResult
	Error  string
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
	Price    *float64 `json:"price"`
	// Code is whatever product code/SKU is printed on the receipt itself —
	// not matched against our own ingredients table.
	Code *string `json:"code"`
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

// receiptPrompt asks Gemini to read a photographed grocery receipt and
// extract what was bought as JSON matching receiptSchema. Can/bottle deposit
// lines are explicitly excluded (they aren't a purchased item); the schema
// only constrains shape/types, not this kind of semantic judgment call, so
// it still needs spelling out here.
const receiptPrompt = `This image is a photo of a grocery store receipt. ` +
	`Extract every item that was purchased: its name, the quantity bought ` +
	`(if shown), the price paid (if shown), and any product code/SKU ` +
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

	data.Result = &result
	h.renderScanReceiptPage(w, r, data)
}

func (h *Handler) renderScanReceiptPage(w http.ResponseWriter, r *http.Request, data ScanReceiptPageData) {
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
