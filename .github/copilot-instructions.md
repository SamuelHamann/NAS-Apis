# Copilot instructions for NAS-Apis

This repository hosts multiple APIs that are deployed on a personal NAS. Each API
lives in its own top-level folder (e.g. `whatsfordinner/`).

## 📌 Always keep documentation in sync

**Whenever a new API is added, or a new route/endpoint is implemented or changed,
you MUST update the documentation in the same change:**

1. **Top-level [`README.md`](../README.md)** — maintain the list of APIs and a
   summary of their endpoints.
2. **The API's own `README.md`** (e.g. `whatsfordinner/README.md`) — update its
   "API reference" table and any relevant configuration/usage notes.

Do this automatically and proactively — do not wait to be asked. Documentation
updates are part of the definition of done for any API or route change.

### When adding a new route, update the API reference table(s)

Use a Markdown table with this shape:

| Method | Path      | Description                     |
| ------ | --------- | ------------------------------- |
| GET    | `/health` | Liveness/readiness health check |

### When adding a brand-new API

- Add it to the top-level `README.md` API list.
- Give it its own folder with a dedicated `README.md` documenting setup,
  configuration, project structure and the endpoint reference.

## Project conventions

- **Go APIs** use the standard library `net/http` with Go 1.22+ routing
  (`mux.HandleFunc("METHOD /path", ...)`). Register all routes in the
  `server` package's `Routes()` function. Keep handlers thin.
- Configuration is read from environment variables (with defaults) in a
  `config` package — avoid hardcoding values.
- Private code lives under `internal/`. The entry point lives in `cmd/<app>/`.
- Each service ships a multi-stage `Dockerfile` so it can be deployed to the NAS.
- Keep changes small and idiomatic for the language in use.
