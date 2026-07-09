package handlers

import (
	"reflect"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string     { return &s }

func TestOrderRecipeIngredients(t *testing.T) {
	// Input arrives alphabetical (as ListRecipeIngredients returns it):
	// Apple (missing), Butter (present), Carrot (missing), Flour (present).
	rows := []store.RecipeIngredientDetailRow{
		{IngredientID: 1, IngredientName: "Apple", Missing: true},
		{IngredientID: 2, IngredientName: "Butter", Missing: false},
		{IngredientID: 3, IngredientName: "Carrot", Missing: true},
		{IngredientID: 4, IngredientName: "Flour", Missing: false},
	}

	got := OrderRecipeIngredients(rows)

	wantOrder := []string{"Apple", "Carrot", "Butter", "Flour"}
	if len(got) != len(wantOrder) {
		t.Fatalf("expected %d rows, got %d", len(wantOrder), len(got))
	}
	for i, name := range wantOrder {
		if got[i].IngredientName != name {
			t.Errorf("index %d: expected %q, got %q", i, name, got[i].IngredientName)
		}
	}

	// Missing ones must come first and be flagged.
	if !got[0].Missing || !got[1].Missing {
		t.Error("expected the first two rows to be flagged Missing")
	}
	if got[2].Missing || got[3].Missing {
		t.Error("expected the last two rows not to be flagged Missing")
	}
}

func TestOrderRecipeIngredientsEmpty(t *testing.T) {
	if got := OrderRecipeIngredients(nil); len(got) != 0 {
		t.Errorf("expected an empty result for nil input, got %v", got)
	}
}

func TestFormatIngredientQuantity(t *testing.T) {
	tests := []struct {
		name string
		row  store.RecipeIngredientDetailRow
		want string
	}{
		{
			name: "no quantity",
			row:  store.RecipeIngredientDetailRow{},
			want: "",
		},
		{
			name: "quantity without unit",
			row:  store.RecipeIngredientDetailRow{Quantity: floatPtr(2)},
			want: "2",
		},
		{
			name: "quantity with unit",
			row:  store.RecipeIngredientDetailRow{Quantity: floatPtr(2), UnitName: strPtr("cups")},
			want: "2 cups",
		},
		{
			name: "fractional quantity trims trailing zeroes",
			row:  store.RecipeIngredientDetailRow{Quantity: floatPtr(0.25), UnitName: strPtr("tsp")},
			want: "0.25 tsp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatIngredientQuantity(tc.row); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestParseInstructionSteps(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "empty",
			raw:  "",
			want: nil,
		},
		{
			name: "numbered with dots",
			raw:  "1. Preheat the oven\n2. Mix the batter\n3. Bake for 20 minutes",
			want: []string{"Preheat the oven", "Mix the batter", "Bake for 20 minutes"},
		},
		{
			name: "numbered with parens and colons",
			raw:  "(1) Preheat the oven\n2: Mix the batter\n3) Bake for 20 minutes",
			want: []string{"Preheat the oven", "Mix the batter", "Bake for 20 minutes"},
		},
		{
			name: "dashed",
			raw:  "- Preheat the oven\n- Mix the batter\n- Bake for 20 minutes",
			want: []string{"Preheat the oven", "Mix the batter", "Bake for 20 minutes"},
		},
		{
			name: "bulleted with asterisks and dots",
			raw:  "* Preheat the oven\n• Mix the batter",
			want: []string{"Preheat the oven", "Mix the batter"},
		},
		{
			name: "plain line breaks with no markers",
			raw:  "Preheat the oven\nMix the batter\nBake for 20 minutes",
			want: []string{"Preheat the oven", "Mix the batter", "Bake for 20 minutes"},
		},
		{
			name: "blank lines are dropped",
			raw:  "1. Preheat the oven\n\n\n2. Mix the batter\n",
			want: []string{"Preheat the oven", "Mix the batter"},
		},
		{
			name: "mixed formatting handled per line",
			raw:  "1. Preheat the oven\n- Mix the batter\nBake for 20 minutes",
			want: []string{"Preheat the oven", "Mix the batter", "Bake for 20 minutes"},
		},
		{
			name: "CRLF line endings",
			raw:  "1. Preheat the oven\r\n2. Mix the batter\r\n",
			want: []string{"Preheat the oven", "Mix the batter"},
		},
		{
			name: "single line, no breaks at all",
			raw:  "Just cook it until done",
			want: []string{"Just cook it until done"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseInstructionSteps(tc.raw)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}
