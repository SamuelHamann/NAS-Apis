package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/units"
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

// recipeIngredientCookRow is one recipe_ingredients row plus its unit's
// display name (joined in, unlike models.RecipeIngredient) — CookRecipe
// needs the name to convert into whatever unit the pantry stock happens to
// be tracked in, via internal/units.
type recipeIngredientCookRow struct {
	RecipeID     int64    `db:"recipe_id"`
	IngredientID int64    `db:"ingredient_id"`
	Quantity     *float64 `db:"quantity"`
	UnitID       *int64   `db:"unit_id"`
	UnitName     *string  `db:"unit_name"`
	Note         *string  `db:"note"`
}

// pantryStockRow is one pantry_ingredients row's quantity/unit, joined with
// its unit's display name — see recipeIngredientCookRow.
type pantryStockRow struct {
	IngredientID int64   `db:"ingredient_id"`
	Quantity     float64 `db:"quantity"`
	UnitName     string  `db:"unit_name"`
}

// CookRecipe performs every side effect of "cooking" a recipe, atomically:
//
//  1. For each of the recipe's ingredients (recipe_ingredients), decrements
//     PantryID's stock by quantity * Multiplier, floored at zero (never goes
//     negative). The decrement is converted from the recipe ingredient's own
//     unit into whatever unit the pantry stock is tracked in (see
//     internal/units) — so a recipe calling for "2 cups" of milk correctly
//     decrements a pantry row tracked in liters, for example. When the two
//     units can't be converted (an unrecognized unit, or a mismatched kind —
//     e.g. volume vs mass), the raw quantity is decremented as a best-effort
//     fallback, same as before unit-aware conversion existed. Ingredients
//     with no specified quantity, or with no matching pantry_ingredients row
//     in that pantry, are left untouched.
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
		SELECT ri.recipe_id, ri.ingredient_id, ri.quantity, ri.unit_id, u.name AS unit_name, ri.note
		FROM   recipe_ingredients ri
		LEFT JOIN units u ON u.id = ri.unit_id
		WHERE  ri.recipe_id = $1`, in.RecipeID)
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	ingredients, err := pgx.CollectRows(rows, pgx.RowToStructByName[recipeIngredientCookRow])
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}

	ingredientIDs := make([]int64, 0, len(ingredients))
	for _, ri := range ingredients {
		ingredientIDs = append(ingredientIDs, ri.IngredientID)
	}
	stockRows, err := tx.Query(ctx, `
		SELECT p.ingredient_id AS ingredient_id, p.quantity AS quantity, u.name AS unit_name
		FROM   pantry_ingredients p
		JOIN   units u ON u.id = p.unit_id
		WHERE  p.pantry_id = $1 AND p.ingredient_id = ANY($2::bigint[])`, in.PantryID, ingredientIDs)
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	stock, err := pgx.CollectRows(stockRows, pgx.RowToStructByName[pantryStockRow])
	if err != nil {
		return CookRecipeResult{}, mapError(err)
	}
	stockByIngredient := make(map[int64]pantryStockRow, len(stock))
	for _, row := range stock {
		stockByIngredient[row.IngredientID] = row
	}

	for _, ri := range ingredients {
		if ri.Quantity == nil {
			continue
		}
		stockRow, ok := stockByIngredient[ri.IngredientID]
		if !ok {
			continue // nothing stocked for this ingredient in this pantry
		}

		amount := *ri.Quantity * in.Multiplier
		decrement := amount
		if ri.UnitName != nil {
			if converted, ok := units.Convert(amount, *ri.UnitName, stockRow.UnitName); ok {
				decrement = converted
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE pantry_ingredients
			SET    quantity = GREATEST(quantity - $3, 0), updated_at = now()
			WHERE  pantry_id = $1 AND ingredient_id = $2`,
			in.PantryID, ri.IngredientID, decrement); err != nil {
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
