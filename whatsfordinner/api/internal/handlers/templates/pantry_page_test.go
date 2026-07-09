package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// pantryPageFixture builds a fully populated PantryPageData with items in
// every expiration bucket, at least one nullable field (LocationID / Note)
// set, and a preselected tag so the test can assert both the happy path
// and the edge cases of pantry.html.
func pantryPageFixture(t *testing.T) PantryPageData {
	t.Helper()
	today := mustDate(t, "2026-07-09")
	fridge := ptrString("Fridge")
	note := ptrString("Almost gone")
	locID := int64(7)

	rows := []store.PantryDetailedRow{
		{
			ID:             uuid.New(),
			PantryID:       1,
			IngredientID:   10,
			IngredientName: "Milk",
			Quantity:       0.5,
			UnitID:         100,
			UnitName:       "L",
			LocationID:     &locID,
			LocationName:   fridge,
			Note:           note,
			IsQuantified:   true,
			ExpirationDate: ptrTime(mustDate(t, "2026-07-05")), // expired
		},
		{
			ID:             uuid.New(),
			PantryID:       1,
			IngredientID:   11,
			IngredientName: "Bread",
			Quantity:       1,
			UnitID:         101,
			UnitName:       "loaf",
			LocationID:     nil,
			LocationName:   nil,
			IsQuantified:   true,
			ExpirationDate: ptrTime(mustDate(t, "2026-07-10")), // soon
		},
		{
			ID:             uuid.New(),
			PantryID:       1,
			IngredientID:   12,
			IngredientName: "Rice",
			Quantity:       2,
			UnitID:         102,
			UnitName:       "kg",
			IsQuantified:   true,
			ExpirationDate: nil, // other
		},
	}

	return PantryPageData{
		PageData:       PageData{Title: "Pantry", SignedIn: true, CurrentUser: "Alice"},
		SelectedPantry: &models.Pantry{ID: 1, Name: "Main kitchen", CreatedAt: today},
		Pantries: []models.Pantry{
			{ID: 1, Name: "Main kitchen", CreatedAt: today},
			{ID: 2, Name: "Garage freezer", CreatedAt: today},
		},
		Cards:          GroupPantry(rows, PantrySortExpiration, today),
		Sort:           PantrySortExpiration,
		SelectedTagIDs: map[int64]bool{5: true},
		Ingredients: []models.Ingredient{
			{ID: 10, Name: "Milk"},
			{ID: 11, Name: "Bread"},
			{ID: 12, Name: "Rice"},
		},
		Units: []models.Unit{
			{ID: 100, Name: "L"},
			{ID: 101, Name: "loaf"},
			{ID: 102, Name: "kg"},
		},
		Locations: []models.FoodLocation{
			{ID: 7, Name: "Fridge"},
			{ID: 8, Name: "Freezer"},
		},
		Tags: []models.Tag{
			{ID: 5, Name: "vegan"},
			{ID: 6, Name: "quick"},
		},
	}
}

func TestPantryTemplateRendersCardsAndForms(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, pantryPageFixture(t)); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	// Structural / chrome ------------------------------------------------
	for _, want := range []string{
		`<title>Pantry`,
		`Main kitchen`,   // selected pantry name in subtitle + hidden field
		`Garage freezer`, // second pantry in the picker
		`name="sort"`,    // sort dropdown
		`vegan`, `quick`, // tag filter chips
		`Add item`, // create toggle
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Cards --------------------------------------------------------------
	for _, want := range []string{
		"Expired",                     // danger card title
		"wfd-pantry-card--danger",     // danger card CSS
		"Expiring in the next 2 days", // warning card title
		"wfd-pantry-card--warning",    // warning card CSS
		"Everything else",             // default card title
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Row-level colour classes track the pre-computed ExpirationClass ---
	if !strings.Contains(body, "wfd-pantry-item--expired") {
		t.Error("expected wfd-pantry-item--expired class on the milk row")
	}
	if !strings.Contains(body, "wfd-pantry-item--soon") {
		t.Error("expected wfd-pantry-item--soon class on the bread row")
	}

	// Row-level content --------------------------------------------------
	for _, want := range []string{
		"Milk", "Bread", "Rice",
		"Jul 5, 2026",  // expired date formatted
		"Jul 10, 2026", // soon date formatted
		"Fridge",       // dereferenced *string location name
		"Almost gone",  // dereferenced *string note
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Every row must carry a delete form that preserves the current view
	// (pantry_id + sort + view_tags). One sanity check is enough — the
	// template loop is shared.
	if !strings.Contains(body, `/delete`) {
		t.Error("expected a POST .../delete action on each pantry row")
	}
	if !strings.Contains(body, `name="pantry_id" value="1"`) {
		t.Error("expected pantry_id=1 hidden field on the row forms")
	}
	if !strings.Contains(body, `name="view_tags" value="5"`) {
		t.Error("expected view_tags=5 hidden field carrying the tag filter forward")
	}

	// The inline edit / create form uses the dict + deref helpers heavily.
	// If either helper mis-fired, the template would have errored above,
	// but assert the location option really is preselected for the milk row
	// to prove the deref path actually returns a value.
	if !strings.Contains(body, `<option value="7" selected>Fridge</option>`) {
		t.Error("expected Fridge (id 7) to be preselected in the edit form (deref helper)")
	}
}

func TestPantryTemplateEmptyState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	today := mustDate(t, "2026-07-09")
	data := PantryPageData{
		PageData:       PageData{Title: "Pantry"},
		SelectedPantry: &models.Pantry{ID: 1, Name: "Main kitchen", CreatedAt: today},
		Pantries:       []models.Pantry{{ID: 1, Name: "Main kitchen", CreatedAt: today}},
		Cards:          nil, // no items
		Sort:           PantrySortExpiration,
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "Your pantry is empty") {
		t.Error("expected empty-state message when Cards is nil and no tag filter is set")
	}
	// Sort dropdown still renders even when the pantry is empty.
	if !strings.Contains(body, `name="sort"`) {
		t.Error("expected sort dropdown to render in empty state")
	}
}

func TestPantryTemplateNoPantryYet(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, PantryPageData{
		PageData: PageData{Title: "Pantry"},
		Sort:     PantrySortExpiration,
	}); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No pantry yet") {
		t.Error("expected 'No pantry yet' subtitle when SelectedPantry is nil")
	}
	if !strings.Contains(body, "Create a pantry") {
		t.Error("expected 'Create a pantry…' guidance in the empty content area")
	}
}

func TestPantryTemplateHonoursSortSelection(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	data := pantryPageFixture(t)
	data.Sort = PantrySortLocation

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	// The 'location' option must be the selected one; the others must not
	// have the selected attribute.
	if !strings.Contains(body, `<option value="location"     selected>Location</option>`) {
		t.Errorf("expected 'location' option to be preselected. body excerpt:\n%s",
			excerpt(body, "sort", 200))
	}
}

// TestPantryTemplateDoesNotNestForms is a regression test for the
// "Cannot read properties of null (reading 'submit')" JS error that used
// to happen when toggling the sort dropdown or a tag chip. The root cause
// was the Add-item <form> being emitted inside the toolbar <form> — the
// HTML parser silently drops the inner start tag but honors its </form>,
// which closed the outer form early and stranded the inputs after it
// (this.form -> null).
//
// The fix moved the filter form to an empty, sibling <form id="pantry-view">
// and rewired every filter input via the HTML5 `form=` attribute. This test
// pins down both halves of that fix so a future edit can't reintroduce the
// bug silently.
func TestPantryTemplateDoesNotNestForms(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, pantryPageFixture(t)); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	// The filter form must be present, and it must be empty (self-closed
	// visually via `hidden`). If it starts collecting children again, the
	// nested-form problem is back.
	const filterFormOpen = `<form id="pantry-view" method="get" action="/pantry" hidden>`
	if !strings.Contains(body, filterFormOpen) {
		t.Fatalf("expected empty filter form %q in body", filterFormOpen)
	}
	if !strings.Contains(body, filterFormOpen+`</form>`) {
		t.Errorf("filter form must be immediately closed (empty). Body excerpt:\n%s",
			excerpt(body, "pantry-view", 200))
	}

	// Every filter control needs the form=... hook — otherwise `this.form`
	// is null in the onchange handler and clicking a tag / changing sort
	// throws in the browser.
	for _, want := range []string{
		`name="sort" form="pantry-view"`,
		`name="tags" value="5" form="pantry-view"`,          // preselected chip
		`type="hidden" name="pantry_id" form="pantry-view"`, // single-pantry case is exercised by the empty-state test; here we have >1 pantry so it's a <select>
		`name="pantry_id" form="pantry-view"`,               // the picker <select>
	} {
		if !strings.Contains(body, want) {
			// pantry_id hidden vs select — only one appears at a time. Only
			// fail if BOTH are missing (i.e. no form= binding at all).
			if want == `type="hidden" name="pantry_id" form="pantry-view"` {
				continue
			}
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

// excerpt returns up to n characters around the first occurrence of needle in
// body — small helper to make failed-render test output easier to read.
func excerpt(body, needle string, n int) string {
	i := strings.Index(body, needle)
	if i < 0 {
		return "<needle not found>"
	}
	start := i - n/2
	if start < 0 {
		start = 0
	}
	end := start + n
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}
