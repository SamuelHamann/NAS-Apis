package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// testPantry is a stand-in SelectedPantry for scan_receipt.html tests: the
// Scan tab's photo-upload form only renders once a pantry is selected (see
// loadPantryPicker in scan_receipt.go), since every queued item needs one.
var testPantry = &models.Pantry{ID: 1, Name: "Main kitchen"}

func TestScanReceiptTemplateRendersForm(t *testing.T) {
	ts, ok := parseTestTemplates(t)["scan_receipt.html"]
	if !ok {
		t.Fatal("scan_receipt.html not found in template cache")
	}

	data := ScanReceiptPageData{
		PageData:       PageData{Title: "Scan receipt"},
		Pantries:       []models.Pantry{*testPantry},
		SelectedPantry: testPantry,
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute scan_receipt.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Scan receipt",
		`action="/scan-receipt"`,
		`enctype="multipart/form-data"`,
		`name="photo"`,
		`capture="environment"`,
		`name="pantry_id" value="1"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
	if strings.Contains(body, `class="wfd-ingredient-list"`) {
		t.Error("expected no result list before a photo has been scanned")
	}
}

func TestScanReceiptTemplateRendersNoPantryState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["scan_receipt.html"]
	if !ok {
		t.Fatal("scan_receipt.html not found in template cache")
	}

	data := ScanReceiptPageData{PageData: PageData{Title: "Scan receipt"}}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute scan_receipt.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "Create a pantry in the database first") {
		t.Error("expected the no-pantry empty state to render")
	}
	if strings.Contains(body, `name="photo"`) {
		t.Error("expected no photo-upload form with no pantry to attach items to")
	}
}

func TestScanReceiptTemplateRendersResult(t *testing.T) {
	ts, ok := parseTestTemplates(t)["scan_receipt.html"]
	if !ok {
		t.Fatal("scan_receipt.html not found in template cache")
	}

	data := ScanReceiptPageData{
		PageData:       PageData{Title: "Scan receipt"},
		Pantries:       []models.Pantry{*testPantry},
		SelectedPantry: testPantry,
		Result: &ReceiptScanResult{
			Items: []ReceiptItem{
				{Name: "Milk", Quantity: floatPtr(1), Price: floatPtr(3.50), Code: strPtr("12345")},
				{Name: "Bread", Quantity: floatPtr(2), Price: floatPtr(5)},
			},
			Total: floatPtr(8.50),
			Taxes: []ReceiptTax{{Name: "GST", Amount: 0.43}},
		},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute scan_receipt.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Milk", "Bread",
		"Code: 12345",
		"$8.5", // Total
		"GST", "0.43",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body, got:\n%s", want, body)
		}
	}
}

func TestScanReceiptTemplateRendersEmptyItems(t *testing.T) {
	ts, ok := parseTestTemplates(t)["scan_receipt.html"]
	if !ok {
		t.Fatal("scan_receipt.html not found in template cache")
	}

	data := ScanReceiptPageData{
		PageData:       PageData{Title: "Scan receipt"},
		Pantries:       []models.Pantry{*testPantry},
		SelectedPantry: testPantry,
		Result:         &ReceiptScanResult{},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute scan_receipt.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No items could be read from this receipt.") {
		t.Error("expected the empty-items message to render")
	}
}

func TestScanReceiptTemplateRendersError(t *testing.T) {
	ts, ok := parseTestTemplates(t)["scan_receipt.html"]
	if !ok {
		t.Fatal("scan_receipt.html not found in template cache")
	}

	data := ScanReceiptPageData{
		PageData: PageData{Title: "Scan receipt"},
		Error:    "please choose or take a photo",
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute scan_receipt.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "please choose or take a photo") {
		t.Error("expected the error banner to render")
	}
}
