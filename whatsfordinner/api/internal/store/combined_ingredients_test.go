package store

import (
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

func floatPtr(f float64) *float64 { return &f }
func int64Ptr(i int64) *int64     { return &i }
func strPtr(s string) *string     { return &s }

func TestMergeCombinedIngredientItemsSumsMatchingIngredients(t *testing.T) {
	existing := []models.CombinedIngredientItem{
		{IngredientID: 1, Quantity: floatPtr(2), UnitID: int64Ptr(10), Note: strPtr("chopped")},
	}
	newItems := []CombinedIngredientItemInput{
		{IngredientID: 1, Quantity: floatPtr(3), UnitID: int64Ptr(20)},
	}

	merged := mergeCombinedIngredientItems(existing, newItems)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged item, got %d", len(merged))
	}
	if merged[0].Quantity == nil || *merged[0].Quantity != 5 {
		t.Errorf("expected summed quantity 5, got %v", merged[0].Quantity)
	}
	// Existing unit/note win when already set, even though the new row
	// specified a different unit and no note.
	if merged[0].UnitID == nil || *merged[0].UnitID != 10 {
		t.Errorf("expected existing unit_id 10 to be kept, got %v", merged[0].UnitID)
	}
	if merged[0].Note == nil || *merged[0].Note != "chopped" {
		t.Errorf("expected existing note to be kept, got %v", merged[0].Note)
	}
}

func TestMergeCombinedIngredientItemsAppendsNewIngredients(t *testing.T) {
	existing := []models.CombinedIngredientItem{
		{IngredientID: 1, Quantity: floatPtr(2)},
	}
	newItems := []CombinedIngredientItemInput{
		{IngredientID: 2, Quantity: floatPtr(4), UnitID: int64Ptr(30), Note: strPtr("to taste")},
	}

	merged := mergeCombinedIngredientItems(existing, newItems)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged items, got %d", len(merged))
	}
	if merged[0].IngredientID != 1 {
		t.Errorf("expected existing ingredient first, got %+v", merged[0])
	}
	if merged[1].IngredientID != 2 || merged[1].Quantity == nil || *merged[1].Quantity != 4 {
		t.Errorf("expected appended ingredient 2 with quantity 4, got %+v", merged[1])
	}
}

func TestMergeCombinedIngredientItemsNilQuantityTakesPresentOne(t *testing.T) {
	existing := []models.CombinedIngredientItem{
		{IngredientID: 1, Quantity: nil},
	}
	newItems := []CombinedIngredientItemInput{
		{IngredientID: 1, Quantity: floatPtr(1.5)},
	}

	merged := mergeCombinedIngredientItems(existing, newItems)
	if len(merged) != 1 || merged[0].Quantity == nil || *merged[0].Quantity != 1.5 {
		t.Fatalf("expected quantity 1.5, got %+v", merged)
	}
}

func TestMergeCombinedIngredientItemsBothNilQuantityStaysNil(t *testing.T) {
	existing := []models.CombinedIngredientItem{
		{IngredientID: 1, Quantity: nil},
	}
	newItems := []CombinedIngredientItemInput{
		{IngredientID: 1, Quantity: nil},
	}

	merged := mergeCombinedIngredientItems(existing, newItems)
	if len(merged) != 1 || merged[0].Quantity != nil {
		t.Fatalf("expected quantity to stay nil, got %+v", merged)
	}
}
