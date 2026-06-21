/**
 * App entry point.
 *
 * Builds the top navigation, instantiates the router and registers every page.
 */

import { Router } from "./router";
import { h, mount } from "./dom";

import { renderHome } from "./pages/home";
import { renderRecipes } from "./pages/recipes";
import { renderIngredients } from "./pages/ingredients";
import { renderUnits } from "./pages/units";
import { renderTags } from "./pages/tags";
import { renderLocations } from "./pages/locations";
import { renderPantry } from "./pages/pantry";
import { renderPastCooked } from "./pages/pastCooked";
import { renderSettings } from "./pages/settings";
import { renderNotFound } from "./pages/notFound";

const NAV_LINKS: Array<[string, string]> = [
  ["/", "Home"],
  ["/recipes", "Recipes"],
  ["/ingredients", "Ingredients"],
  ["/units", "Units"],
  ["/tags", "Tags"],
  ["/locations", "Locations"],
  ["/pantry", "Pantry"],
  ["/past-cooked", "History"],
  ["/settings", "Settings"],
];

function renderNav(): void {
  mount(
    "#nav",
    ...NAV_LINKS.map(([href, label]) =>
      h("a", { href: `#${href}` }, label),
    ),
  );
}

renderNav();

const router = new Router("#app");
router
  .add("/", renderHome)
  .add("/recipes", renderRecipes)
  .add("/ingredients", renderIngredients)
  .add("/units", renderUnits)
  .add("/tags", renderTags)
  .add("/locations", renderLocations)
  .add("/pantry", renderPantry)
  .add("/past-cooked", renderPastCooked)
  .add("/settings", renderSettings)
  .setNotFound((params) => renderNotFound(params))
  .start();
