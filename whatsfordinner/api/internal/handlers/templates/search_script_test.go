package handlers

import (
	"bytes"
	"strings"
	"testing"
)

// TestPantryTemplateRendersSearchBox checks the markup that
// templates/scripts.html's live "type to filter" JS depends on: the search
// input's wiring attributes, the id'd cards container it points at, a
// data-search-card on every card, a data-search-name on every item (set to
// the ingredient name), and the empty-state paragraph.
func TestPantryTemplateRendersSearchBox(t *testing.T) {
	ts, ok := parseTestTemplates(t)["pantry.html"]
	if !ok {
		t.Fatal("pantry.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, pantryPageFixture(t)); err != nil {
		t.Fatalf("execute pantry.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		`data-search-input="#pantry-cards"`,
		`data-search-empty="#pantry-search-empty"`,
		`id="pantry-cards"`,
		`id="pantry-search-empty"`,
		"data-search-card",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Every item row must carry its ingredient name for the JS to match
	// against — spot-check with the fixture's known ingredients.
	for _, want := range []string{
		`data-search-name="Milk"`,
		`data-search-name="Bread"`,
		`data-search-name="Rice"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

// TestRecipesTemplateRendersSearchBox is the recipe-page counterpart of
// TestPantryTemplateRendersSearchBox.
func TestRecipesTemplateRendersSearchBox(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, recipesPageFixture(t)); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		`data-search-input="#recipes-cards"`,
		`data-search-empty="#recipes-search-empty"`,
		`id="recipes-cards"`,
		`id="recipes-search-empty"`,
		"data-search-card",
		`data-search-name="Pasta al pomodoro"`,
		`data-search-name="Chicken curry"`,
		`data-search-name="Sushi platter"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

// TestStylesForceHiddenAttributeDisplayNone is a regression guard for a bug
// where the live search filter (which toggles the `hidden` attribute on
// individual pantry/recipe rows, see templates/scripts.html) appeared to do
// nothing until every row in a card was filtered out and the whole card
// vanished. Root cause: `[hidden] { display: none }` only exists in the
// browser's default stylesheet, which loses to *any* author rule setting
// `display` on the same element — e.g. .wfd-pantry-item/.wfd-status-item's
// `display: flex`. templates/styles.html must re-assert `[hidden]` with
// `!important` so JS-toggled hidden rows actually disappear.
func TestStylesForceHiddenAttributeDisplayNone(t *testing.T) {
	ts, ok := parseTestTemplates(t)["styles"]
	if !ok {
		t.Fatal(`template "styles" not found in template cache`)
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, PageData{Title: "Pantry"}); err != nil {
		t.Fatalf("execute styles: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "[hidden]") || !strings.Contains(body, "display: none !important") {
		t.Error("expected styles.html to force [hidden] { display: none !important } " +
			"so JS-toggled hidden rows (e.g. the search filter) actually disappear " +
			"even on elements with a competing `display: flex/grid` rule")
	}
}

// TestSearchScriptTemplateRenders confirms the shared search-script
// partial parses and executes cleanly (it takes no meaningful data, just
// the page context, so any PageData-embedding value works).
func TestSearchScriptTemplateRenders(t *testing.T) {
	ts, ok := parseTestTemplates(t)["search-script"]
	if !ok {
		t.Fatal(`template "search-script" not found in template cache`)
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, PageData{Title: "Pantry"}); err != nil {
		t.Fatalf("execute search-script: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"<script>",
		"data-search-input",
		"data-search-name",
		"data-search-card",
		"data-search-empty",
		"addEventListener('input'",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered search-script output", want)
		}
	}
}
