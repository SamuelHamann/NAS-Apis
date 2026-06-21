# NAS-Apis

A collection of small, self-hosted APIs deployed on my NAS. Each API lives in its
own folder and is independently buildable and deployable.

## APIs

| API                                | Language | Description                                  | Status        |
| ---------------------------------- | -------- | -------------------------------------------- | ------------- |
| [whatsfordinner](./whatsfordinner) | Go       | Helps decide what's for dinner (recipes etc) | 🚧 Basic CRUD |

## Endpoints overview

### whatsfordinner

PostgreSQL-backed CRUD. Each resource below supports the standard set: `GET`
(list), `POST` (create), and `GET` / `PUT` / `DELETE` on `/{id}`.

| Method(s) | Path(s)                           | Description                          |
| --------- | --------------------------------- | ------------------------------------ |
| GET       | `/health`, `/ready`               | Liveness and DB-readiness checks     |
| CRUD      | `/recipes`, `/recipes/{id}`       | Recipes (UUID id)                    |
| GET       | `/recipes/cookable`               | Recipes cookable from pantry stock   |
| CRUD      | `/ingredients`, `/ingredients/{id}` | Canonical ingredients (integer id) |
| CRUD      | `/units`, `/units/{id}`           | Measurement units (integer id)       |
| CRUD      | `/locations`, `/locations/{id}`   | Food storage locations (integer id)  |
| CRUD      | `/tags`, `/tags/{id}`             | Recipe tags (integer id)             |
| CRUD      | `/pantry`, `/pantry/{id}`         | Home pantry stock (UUID id)          |
| CRUD      | `/past-cooked`, `/past-cooked/{id}` | Cooking history (UUID id)          |

See [`whatsfordinner/README.md`](./whatsfordinner/README.md) for full details,
request bodies and configuration.

## Conventions

- Each API is a self-contained project in its own top-level folder.
- Each API documents its own setup, configuration and endpoints in a local
  `README.md`, and is summarized in the tables above.
- APIs are containerized for deployment on the NAS.

> **Note:** Whenever an API or route is added or changed, update this README and
> the relevant API's README. See [`.github/copilot-instructions.md`](./.github/copilot-instructions.md).
