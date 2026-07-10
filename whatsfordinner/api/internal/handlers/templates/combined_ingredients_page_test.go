package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// combinedIngredientsPageFixture extends ingredientsPageFixture with the
// combined-ingredients tab's data: one bundle with two component items and
// one bundle with zero items (to exercise the "always >=1 starter row" edit
// form fallback), plus the Ingredients/Units dropdown data both tabs' forms
// need.
func combinedIngredientsPageFixture() IngredientsPageData {
	data := ingredientsPageFixture()
	data.Ingredients = []models.Ingredient{
		{ID: 1, Name: "Paprika"},
		{ID: 2, Name: "Cumin"},
		{ID: 3, Name: "Salt"},
	}
	data.Units = []models.Unit{
		{ID: 20, Name: "tbsp"},
		{ID: 21, Name: "tsp"},
	}
	unitID := int64(20)
	quantity := 2.0
	itemUnitID := int64(21)
	data.CombinedItems = []store.CombinedIngredientDetailRow{
		{
			ID: 100, Name: "Taco Seasoning Mix", Quantity: 3, UnitID: &unitID, UnitName: strPtr("tbsp"),
			Items: []store.CombinedIngredientItemDetailRow{
				{IngredientID: 1, IngredientName: "Paprika", Quantity: &quantity, UnitID: &itemUnitID, UnitName: strPtr("tsp")},
				{IngredientID: 3, IngredientName: "Salt"},
			},
		},
		{ID: 101, Name: "Empty Bundle", Quantity: 1, UnitID: &unitID, UnitName: strPtr("tbsp")},
	}
	return data
}

func TestIngredientsTemplateRendersCombinedTab(t *testing.T) {
	ts, ok := parseTestTemplates(t)["ingredients.html"]
	if !ok {
		t.Fatal("ingredients.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, combinedIngredientsPageFixture()); err != nil {
		t.Fatalf("execute ingredients.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Ingredients</a>", "Combined ingredients</a>",
		`id="combined-panel"`,
		"Taco Seasoning Mix", "Empty Bundle",
		"Paprika", "Salt",
		`action="/combined-ingredients"`,
		`action="/combined-ingredients/100/update"`,
		`action="/combined-ingredients/100/delete"`,
		`name="item_ingredient_id"`,
		`name="item_quantity"`,
		`name="item_unit_id"`,
		`name="item_note"`,
		"<template data-item-row-template>",
		`data-item-row-add`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestIngredientsTemplateDefaultsToIngredientsTab(t *testing.T) {
	ts, ok := parseTestTemplates(t)["ingredients.html"]
	if !ok {
		t.Fatal("ingredients.html not found in template cache")
	}

	data := combinedIngredientsPageFixture()
	data.ActiveTab = ""

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute ingredients.html: %v", err)
	}
	body := buf.String()

	ingredientsPanelIdx := strings.Index(body, `id="ingredients-panel"`)
	combinedPanelIdx := strings.Index(body, `id="combined-panel"`)
	if ingredientsPanelIdx == -1 || combinedPanelIdx == -1 {
		t.Fatal("expected both panel ids to render")
	}

	ingredientsPanelTag := body[ingredientsPanelIdx : ingredientsPanelIdx+120]
	combinedPanelTag := body[combinedPanelIdx : combinedPanelIdx+120]

	if strings.Contains(ingredientsPanelTag, "hidden") {
		t.Error("expected the ingredients panel to be visible by default")
	}
	if !strings.Contains(combinedPanelTag, "hidden") {
		t.Error("expected the combined panel to be hidden by default")
	}
}

func TestIngredientsTemplateShowsCombinedTabWhenActive(t *testing.T) {
	ts, ok := parseTestTemplates(t)["ingredients.html"]
	if !ok {
		t.Fatal("ingredients.html not found in template cache")
	}

	data := combinedIngredientsPageFixture()
	data.ActiveTab = "combined"

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute ingredients.html: %v", err)
	}
	body := buf.String()

	ingredientsPanelIdx := strings.Index(body, `id="ingredients-panel"`)
	combinedPanelIdx := strings.Index(body, `id="combined-panel"`)
	if ingredientsPanelIdx == -1 || combinedPanelIdx == -1 {
		t.Fatal("expected both panel ids to render")
	}

	ingredientsPanelTag := body[ingredientsPanelIdx : ingredientsPanelIdx+120]
	combinedPanelTag := body[combinedPanelIdx : combinedPanelIdx+120]

	if !strings.Contains(ingredientsPanelTag, "hidden") {
		t.Error("expected the ingredients panel to be hidden when combined tab is active")
	}
	if strings.Contains(combinedPanelTag, "hidden") {
		t.Error("expected the combined panel to be visible when combined tab is active")
	}
}

func TestParseCombinedIngredientForm(t *testing.T) {
	form := map[string][]string{
		"name":               {"  Taco Seasoning Mix  "},
		"quantity":           {"3"},
		"unit_id":            {"20"},
		"note":               {"  keep airtight  "},
		"item_ingredient_id": {"1", "3"},
		"item_quantity":      {"2", ""},
		"item_unit_id":       {"21", ""},
		"item_note":          {"", "  to taste  "},
	}

	in, msg := parseCombinedIngredientForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message, got %q", msg)
	}
	if in.Name != "Taco Seasoning Mix" {
		t.Errorf("expected trimmed name, got %q", in.Name)
	}
	if in.Quantity != 3 {
		t.Errorf("expected quantity 3, got %v", in.Quantity)
	}
	if in.UnitID == nil || *in.UnitID != 20 {
		t.Errorf("expected unit_id 20, got %v", in.UnitID)
	}
	if in.Note == nil || *in.Note != "keep airtight" {
		t.Errorf("expected trimmed note, got %v", in.Note)
	}
	if len(in.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(in.Items))
	}
	if in.Items[0].IngredientID != 1 || in.Items[0].Quantity == nil || *in.Items[0].Quantity != 2 || in.Items[0].UnitID == nil || *in.Items[0].UnitID != 21 {
		t.Errorf("unexpected item[0]: %+v", in.Items[0])
	}
	if in.Items[1].IngredientID != 3 || in.Items[1].Quantity != nil || in.Items[1].UnitID != nil || in.Items[1].Note == nil || *in.Items[1].Note != "to taste" {
		t.Errorf("unexpected item[1]: %+v", in.Items[1])
	}
}

func TestParseCombinedIngredientFormRequiresName(t *testing.T) {
	form := map[string][]string{
		"name":               {"  "},
		"quantity":           {"1"},
		"unit_id":            {"20"},
		"item_ingredient_id": {"1"},
	}
	if _, msg := parseCombinedIngredientForm(form); msg == "" {
		t.Error("expected a validation message for a blank name")
	}
}

func TestParseCombinedIngredientFormRequiresQuantity(t *testing.T) {
	form := map[string][]string{
		"name":               {"Mix"},
		"unit_id":            {"20"},
		"item_ingredient_id": {"1"},
	}
	if _, msg := parseCombinedIngredientForm(form); msg == "" {
		t.Error("expected a validation message for a missing quantity")
	}
}

func TestParseCombinedIngredientFormRequiresUnit(t *testing.T) {
	form := map[string][]string{
		"name":               {"Mix"},
		"quantity":           {"1"},
		"item_ingredient_id": {"1"},
	}
	if _, msg := parseCombinedIngredientForm(form); msg == "" {
		t.Error("expected a validation message for a missing unit")
	}
}

func TestParseCombinedIngredientFormSkipsBlankTrailingRow(t *testing.T) {
	form := map[string][]string{
		"name":               {"Mix"},
		"quantity":           {"1"},
		"unit_id":            {"20"},
		"item_ingredient_id": {"1", "3", ""},
		"item_quantity":      {"", "", ""},
		"item_unit_id":       {"", "", ""},
		"item_note":          {"", "", ""},
	}
	in, msg := parseCombinedIngredientForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message, got %q", msg)
	}
	if len(in.Items) != 2 {
		t.Fatalf("expected the all-blank trailing row to be skipped, got %d items", len(in.Items))
	}
}

func TestParseCombinedIngredientFormRejectsDuplicateIngredientRows(t *testing.T) {
	form := map[string][]string{
		"name":               {"Mix"},
		"quantity":           {"1"},
		"unit_id":            {"20"},
		"item_ingredient_id": {"1", "1"},
	}
	if _, msg := parseCombinedIngredientForm(form); msg == "" {
		t.Error("expected a validation message for duplicate component ingredients")
	}
}

func TestParseCombinedIngredientFormRequiresAtLeastOneItem(t *testing.T) {
	form := map[string][]string{
		"name":     {"Mix"},
		"quantity": {"1"},
		"unit_id":  {"20"},
	}
	if _, msg := parseCombinedIngredientForm(form); msg == "" {
		t.Error("expected a validation message when no items are given")
	}
}
