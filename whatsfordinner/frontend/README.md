# whatsfordinner frontend

A small TypeScript single-page app that talks to the
[whatsfordinner API](../README.md). Part of the
[NAS-Apis](../../README.md) collection.

It is deliberately tiny — no framework, no transitive dependency tree to
maintain:

- **TypeScript** (strict) for the code.
- **esbuild** as the bundler and dev server (the only runtime dependency).
- **Vanilla DOM** with a hand-rolled `h()` helper and a hash-based router.

## Quick start

```sh
cd whatsfordinner/frontend
npm install
npm run dev          # http://localhost:5173 (TypeScript dev server, source maps)
```

The API must also be running — see [`../README.md`](../README.md).

On first load, open `#/settings` to set:

- **API base URL** — defaults to `""` (same origin as the page). For
  `npm run dev` against a separately-running API, set it to the API's URL,
  e.g. `http://localhost:8080`.
- **API key** — a UUID from the API's `api_keys` table. Sent as the
  `X-Api-Key` header on every request (except `/health` and `/ready`).

Both values are stored in `localStorage` under the `wfd:` prefix.

## Production deployment

In production, **the SPA is embedded into the Go API binary** via `//go:embed`
(see [`../api/internal/server/static.go`](../api/internal/server/static.go))
and served from the same origin as the API itself. There is no separate web
server, no proxy, and no CORS to manage — just one container.

The build pipeline:

1. `npm run build` (or the Dockerfile's first stage) produces
   `public/dist/app.js` and bundles `public/` as the static SPA root.
2. The Go build copies `public/` into `api/internal/server/web/` and runs
   `go build` — the embed directive includes the bundle in the binary.
3. The binary's HTTP server registers a catch-all SPA handler at `/`. API
   routes (more specific patterns) win for paths like `/recipes`, `/health`;
   everything else falls through to the embedded `index.html` so the in-app
   hash router renders the right view.

Locally, `make spa` from `whatsfordinner/api/` runs the full pipeline.

## Scripts

| Script              | Description                                            |
| ------------------- | ------------------------------------------------------ |
| `npm run dev`       | esbuild dev server, on-demand rebuilds, source maps    |
| `npm run build`     | Minified production bundle into `public/dist/app.js`   |
| `npm run typecheck` | `tsc --noEmit` — type-check without producing any JS   |
| `npm run clean`     | Remove `public/dist/`                                  |

esbuild bundles `src/main.ts` → `public/dist/app.js`. The static shell lives
at `public/index.html` and references `/dist/app.js` and `/styles.css`.

## Project structure

```
frontend/
├── public/
│   ├── index.html          # Static shell, loads /dist/app.js
│   ├── styles.css
│   └── dist/               # Build output (gitignored)
├── src/
│   ├── api/                # One file per resource — typed fetch wrappers
│   │   ├── client.ts       # Base fetch wrapper, attaches X-Api-Key
│   │   ├── health.ts
│   │   ├── recipes.ts
│   │   ├── ingredients.ts
│   │   ├── units.ts
│   │   ├── tags.ts
│   │   ├── locations.ts
│   │   ├── pantry.ts
│   │   └── pastCooked.ts
│   ├── pages/              # One render function per route
│   │   ├── resourceList.ts # Shared "list view" helper
│   │   ├── home.ts
│   │   ├── recipes.ts
│   │   ├── ingredients.ts
│   │   ├── units.ts
│   │   ├── tags.ts
│   │   ├── locations.ts
│   │   ├── pantry.ts
│   │   ├── pastCooked.ts
│   │   ├── settings.ts
│   │   └── notFound.ts
│   ├── types/
│   │   └── models.ts       # TS types mirroring the Go internal/models structs
│   ├── config.ts           # localStorage-backed runtime config
│   ├── dom.ts              # h() / mount() / clear() helpers (no framework)
│   ├── router.ts           # Tiny hash-based router
│   └── main.ts             # Entry point — wires nav + routes
├── .env.example            # Documents the typical config values
├── .gitignore
├── package.json
├── tsconfig.json
└── README.md
```

### What goes where

- **`src/api/`** — A thin, typed wrapper per resource. Every function returns a
  promise and accepts an optional `AbortSignal`. `client.ts` is the only place
  that touches `fetch` and the API key.
- **`src/types/models.ts`** — The single source of truth for the response and
  request shapes. Mirrors `whatsfordinner/api/internal/models/models.go`;
  pointer fields on the Go side become `T | null`.
- **`src/pages/`** — One render function per route. List pages reuse
  `resourceList.ts`. Pages can call into `src/api/*` directly.
- **`src/router.ts`** — `Router` supports literal segments and `:param`
  segments. The matched params are passed to the render function.
- **`src/dom.ts`** — `h(tag, attrs, ...children)` builds elements
  imperatively (no `innerHTML`, no JSX). `mount(selector, ...children)`
  replaces the children of a target element.

## Adding a new page

1. If new endpoints are needed, add them to (or create) a `src/api/<resource>.ts`
   and the matching types in `src/types/models.ts`.
2. Create `src/pages/<name>.ts` exporting a `render…()` function.
3. Register the route in `src/main.ts`:
   ```ts
   router.add("/<path>", render…);
   ```
4. Add the route to `NAV_LINKS` in `src/main.ts` if it should appear in the
   top navigation.
5. **Update the API reference table** in [`../README.md`](../README.md) and the
   top-level [`../../README.md`](../../README.md) if you also touched the
   backend (see [`../../.github/copilot-instructions.md`](../../.github/copilot-instructions.md)).
