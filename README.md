# NAS-Apis

A collection of small, self-hosted APIs deployed on my NAS. Each API lives in its
own folder and is independently buildable and deployable.

## APIs

| API                                | Language       | Description                                  | Status                                |
| ---------------------------------- | -------------- | -------------------------------------------- | ------------------------------------- |
| [whatsfordinner](./whatsfordinner) | Go | Helps decide what's for dinner (recipes etc) | 🚧 Server-rendered HTML UI + JSON API (WIP) |

## Endpoints overview

### whatsfordinner

The UI is being migrated from JSON responses to server-rendered HTML pages
(see [`whatsfordinner/README.md`](./whatsfordinner/README.md)); the
JSON-CRUD endpoints below are still the data API used internally and remain
available while that migration is in progress.

| Method(s) | Path(s)                           | Description                          |
| --------- | --------------------------------- | ------------------------------------ |
| GET       | `/health`, `/ready`               | Liveness and DB-readiness checks     |
| GET       | `/`, `/home`                      | Home page (HTML)                     |
| GET       | `/login`                          | Sign-in / user picker page (HTML)    |
| POST      | `/users`                          | Create a user (HTML form)            |
| POST      | `/users/{id}/update`              | Rename a user (HTML form)            |
| POST      | `/users/{id}/delete`              | Delete a user (HTML form)            |
| POST      | `/users/{id}/select`              | Sign in as this user (sets a cookie) |
| GET       | `/pantry`                         | Pantry page (HTML): grouped/coloured stock with sort + tag filter |
| POST      | `/pantry`                         | Add a pantry item (HTML form)        |
| POST      | `/pantry/{id}/update`             | Edit a pantry item (HTML form)       |
| POST      | `/pantry/{id}/delete`             | Delete a pantry item (HTML form)     |
| GET       | `/recipes`                        | Recipes page (HTML): grouped/coloured by pantry-relative readiness, with sort + tag filter |
| GET       | `/recipes/{id}`                   | Recipe detail page (HTML): ingredients (missing ones highlighted) + instructions |
| POST      | `/recipes/{id}/cook`              | Cook a recipe: decrement pantry stock, log it as cooked, optionally save leftovers as a combined ingredient (HTML form) |
| CRUD      | `/recipes`, `/recipes/{id}`       | Recipes (bigint id)                    |
| GET       | `/recipes/cookable`               | Recipes cookable from pantry stock   |
| GET       | `/ingredients`                    | Ingredients page (HTML): Ingredients tab + Combined ingredients tab (`?tab=combined`), switched client-side |
| CRUD      | `/ingredients`, `/ingredients/{id}` | Canonical ingredients (integer id) |
| POST      | `/combined-ingredients`, `/combined-ingredients/{id}/update`, `/combined-ingredients/{id}/delete` | Combined ingredients: bundles of component ingredients with their own quantity/unit (HTML forms) |
| CRUD      | `/units`, `/units/{id}`           | Measurement units (integer id)       |
| CRUD      | `/locations`, `/locations/{id}`   | Food storage locations (integer id)  |
| CRUD      | `/tags`, `/tags/{id}`             | Recipe tags (integer id)             |
| CRUD      | `/past-cooked`, `/past-cooked/{id}` | Cooking history (UUID id)          |
| GET       | `/scan-receipt`                   | Scan receipt page (HTML, mobile-only entry point): photo-upload form, plus a "Queue" tab listing `pending_pantry_items` filterable by status |
| POST      | `/scan-receipt`                   | Send the photo to Gemini (structured JSON output), render the parsed item list + total/taxes, and queue every item as a `pending_pantry_items` row against the selected pantry (`pending` if a UPC was derived, `rejected` otherwise) (HTML form) |

See [`whatsfordinner/README.md`](./whatsfordinner/README.md) for full details,
request bodies and configuration. The UI (navbar, home page, sign-in) is
rendered server-side with Go's `html/template` and ships inside the API
binary itself — no separate frontend build or container.

whatsfordinner also runs one background job outside the HTTP server: a
queue worker (`internal/worker`) that periodically resolves `pending_pantry_items`
rows against [OpenFoodFacts](https://openfoodfacts.org) by UPC — see
[Queue worker](./whatsfordinner/README.md#queue-worker) in its README.

## Conventions

- Each API is a self-contained project in its own top-level folder.
- Each API documents its own setup, configuration and endpoints in a local
  `README.md`, and is summarized in the tables above.
- APIs are containerized for deployment on the NAS.

> **Note:** Whenever an API or route is added or changed, update this README and
> the relevant API's README. See [`.github/copilot-instructions.md`](./.github/copilot-instructions.md).
