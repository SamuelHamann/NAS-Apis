package handlers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipeIngredientView is one ingredient row on the recipe detail page: the
// store row plus a pre-formatted quantity string so the template never has
// to do arithmetic or figure out whether a unit applies.
type RecipeIngredientView struct {
	store.RecipeIngredientDetailRow
	// QuantityText is e.g. "2 cups" or "" when the recipe doesn't specify
	// an amount for this ingredient.
	QuantityText string
}

// OrderRecipeIngredients arranges a recipe's ingredients for display: every
// ingredient missing from the pantry first (alphabetical), then everything
// already in stock (also alphabetical). rows is expected to already be
// sorted alphabetically by ingredient name — ListRecipeIngredients does
// this in SQL — so this is a stable partition, not a re-sort.
func OrderRecipeIngredients(rows []store.RecipeIngredientDetailRow) []RecipeIngredientView {
	views := make([]RecipeIngredientView, len(rows))
	for i, row := range rows {
		views[i] = RecipeIngredientView{
			RecipeIngredientDetailRow: row,
			QuantityText:              formatIngredientQuantity(row),
		}
	}

	ordered := make([]RecipeIngredientView, 0, len(views))
	for _, v := range views {
		if v.Missing {
			ordered = append(ordered, v)
		}
	}
	for _, v := range views {
		if !v.Missing {
			ordered = append(ordered, v)
		}
	}
	return ordered
}

// formatIngredientQuantity renders "2 cups" / "2" / "" depending on which of
// Quantity/UnitName are present. Trailing zeroes are trimmed (2, not 2.0).
func formatIngredientQuantity(row store.RecipeIngredientDetailRow) string {
	if row.Quantity == nil {
		return ""
	}
	qty := strconv.FormatFloat(*row.Quantity, 'f', -1, 64)
	if row.UnitName == nil || *row.UnitName == "" {
		return qty
	}
	return qty + " " + *row.UnitName
}

// --- Instruction parsing ---------------------------------------------------

var (
	// Matches a numbered-list marker at the start of a line: "1. ", "1) ",
	// "1: ", optionally parenthesised ("(1) ").
	numberedStepRe = regexp.MustCompile(`^\(?\d+[.):]\s+`)
	// Matches a dash/bullet marker at the start of a line: "- ", "* ", "• ".
	dashedStepRe = regexp.MustCompile(`^[-*•]\s+`)
)

// ParseInstructionSteps splits a recipe's free-form Instructions text into
// one entry per step. Recipes get pasted in from all kinds of sources, so
// this looks at each line and tries, in order:
//
//   - Numbered ("1. Preheat the oven" / "1) ..." / "1: ...") — the marker
//     is stripped and the rest of the line becomes the step.
//   - Dashed/bulleted ("- Preheat the oven" / "* ..." / "• ...") — likewise
//     stripped.
//   - Otherwise the line is used as-is, so a recipe that is simply
//     separated by line breaks (no markers at all) still becomes one step
//     per line instead of one giant paragraph.
//
// Blank lines are dropped. Mixed formatting (some numbered, some plain
// lines) is handled per-line rather than requiring the whole recipe to be
// consistent. An empty input yields a nil slice.
func ParseInstructionSteps(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	var steps []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case numberedStepRe.MatchString(line):
			line = numberedStepRe.ReplaceAllString(line, "")
		case dashedStepRe.MatchString(line):
			line = dashedStepRe.ReplaceAllString(line, "")
		}
		if line = strings.TrimSpace(line); line != "" {
			steps = append(steps, line)
		}
	}
	return steps
}
