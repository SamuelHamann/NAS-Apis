package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// CookRecipeInput holds everything needed to "cook" a recipe.
type CookRecipeInput struct {
	RecipeID   int64
	RecipeName string // used to name the auto-created combined ingredient
	PantryID   int64
	Multiplier float64 // e.g. 1.2 for "cook 1.2x the recipe"

	// CreateCombined, when true, folds the recipe's ingredients (each
	// multiplied by Multiplier) into a combined ingredient named after the
	// recipe — see upsertCombinedIngredientFromRecipeTx. CombinedQuantity/
	// CombinedUnitID are that combined ingredient's own top-level
	// quantity/unit (independent of Multiplier, even though the UI defaults
	// the former to it); both are ignored when CreateCombined is false.
	CreateCombined   bool
	CombinedQuantity float64
	CombinedUnitID   *int64
}

// CookRecipeResult is what CookRecipe changed.
type CookRecipeResult struct {
	PastCooked models.PastCookedRecipe
}

// CookRecipe performs every side effect of "cooking" a recipe, atomically:
//
//  1. For each of the recipe's ingredients (recipe_ingredients), decrements
//     PantryID's stock by quantity * Multiplier, floored at zero (never goes
//     negative). Ingredients with no specified quantity, or with no matching
//     pantry_ingredients row in that pantry, are left untouched.
//  2. Upserts past_cooked_recipes for the recipe: times_cooked +1 (starting
//     at 1 the first time) and last_cooked_at = now().
//  3. If CreateCombined, folds the recipe's ingredients into a combined
//     ingredient named after the recipe (creating it, or adding to it if one
//     already exists — see upsertCombinedIngredientFromRecipeTx).
func (s *Store) CookRecipe(ctx context.Context, in CookRecipeInput) (CookRecipeResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT recipe_id, ingredient_id, quantity, unit_id, note
		FROM   recipe_ingredients
		WHERE  recipe_id = $1`, in.RecipeID)
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	ingredients, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.RecipeIngredient])
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}

	for _, ri := range ingredients {
		if ri.Quantity == nil {
			continue
		}
		amount := *ri.Quantity * in.Multiplier
		if _, err := tx.Exec(ctx, `
			UPDATE pantry_ingredients
			SET    quantity = GREATEST(quantity - $3, 0), updated_at = now()
			WHERE  pantry_id = $1 AND ingredient_id = $2`,
			in.PantryID, ri.IngredientID, amount); err != nil {
			return CookRecipeResult{}, mapError(err)
		}
	}

	pastRows, err := tx.Query(ctx, `
		INSERT INTO past_cooked_recipes (recipe_id, times_cooked, last_cooked_at)
		VALUES ($1, 1, now())
		ON CONFLICT (recipe_id) DO UPDATE
			SET times_cooked   = past_cooked_recipes.times_cooked + 1,
			    last_cooked_at = now()
		RETURNING id, recipe_id, times_cooked, last_cooked_at`, in.RecipeID)
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	pastCooked, err := pgx.CollectOneRow(pastRows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}

	if in.CreateCombined {
		items := make([]CombinedIngredientItemInput, 0, len(ingredients))
		for _, ri := range ingredients {
			item := CombinedIngredientItemInput{IngredientID: ri.IngredientID, UnitID: ri.UnitID, Note: ri.Note}
			if ri.Quantity != nil {
				q := *ri.Quantity * in.Multiplier
				item.Quantity = &q
			}
			items = append(items, item)
		}
		if err := upsertCombinedIngredientFromRecipeTx(ctx, tx, in.RecipeName, in.CombinedQuantity, in.CombinedUnitID, items); err != nil {
			return CookRecipeResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	return CookRecipeResult{PastCooked: pastCooked}, nil
}
