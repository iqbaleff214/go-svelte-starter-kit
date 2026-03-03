# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A full-stack starter kit pairing a **Go** backend (Chi router) with a **SvelteKit** frontend (SSR disabled, CSR-only). Currently in Phase 1 (auth scaffold). See `PRD.md` for the full feature roadmap.

## Commands

### First-time Setup
```sh
make setup         # copy .env.example files, install frontend deps, download Go modules
```

### Development
```sh
make infra         # start Postgres + Redis via docker-compose.dev.yml
make dev           # start infra + backend (go run ./cmd/api) + frontend (vite dev) with hot-reload
make stop          # stop all dev infrastructure
```

### Database Migrations
```sh
make migrate                                    # apply all pending migrations
make migrate-down                               # roll back last migration
make migrate-create MIGRATION_NAME=add_users    # create new sequential migration files
make migrate-status                             # show current migration version
```
Migrations run via the `migrate/migrate` Docker image against `DATABASE_URL`.

### Testing
```sh
make test          # run all backend tests (go test ./... -v -count=1)
make test-cover    # run tests with HTML coverage report
make test-fe       # run frontend tests (vitest run)
```
To run a single Go test: `cd backend && go test ./internal/auth/... -run TestServiceLogin -v`

### Linting & Formatting
```sh
make lint          # golangci-lint (backend) + eslint (frontend)
make fmt           # gofmt (backend) + prettier (frontend)
```

### Building
```sh
make build-be      # build backend binaries → backend/bin/api, backend/bin/worker
make build-fe      # vite build (frontend production)
make build         # build production Docker images
```

## Architecture

### Monorepo Layout
```
backend/    Go API + worker
frontend/   SvelteKit app
Makefile    unified dev commands (run from repo root)
migrations/ (inside backend/) sequential SQL files managed by golang-migrate
```

### Backend (`backend/`)

**Entrypoints:**
- `cmd/api/main.go` — HTTP API server
- `cmd/worker/main.go` — background job worker (planned)

**Package conventions (`internal/` vs `pkg/`):**
- `internal/<domain>/` — one directory per feature domain (e.g., `auth/`). Each domain owns `handler.go`, `service.go`, `repository.go`, `model.go`
- `pkg/` — shared infrastructure: `config`, `database`, `redis`, `token` (JWT), `validator`, `logger`

**Request lifecycle:**
1. `pkg/config` loads env vars (panics on missing required vars)
2. `internal/server/server.go` wires all dependencies and registers Chi routes
3. `internal/middleware/` provides `Authenticate` (JWT bearer), `RequireRole`, `RateLimit`, `Logger`
4. Domain handlers call services; services call repositories (direct `pgx` queries — no ORM yet)

**Response format (always follow these):**
- Success: `{ "data": {...}, "meta": {...} }`
- Error: `{ "error": { "code": "snake_case", "message": "Human readable" } }`
- Validation: `{ "error": { "code": "validation_failed", "message": "...", "details": [...] } }`

**Auth flow:**
- Access token: short-lived JWT (15 min) — returned in response body, stored in memory on the frontend
- Refresh token: long-lived JWT (7 days) — set as `HttpOnly; SameSite=Strict` cookie scoped to `/api/auth`
- Refresh rotation: old session revoked, new session created on each refresh
- Token hashing: refresh tokens are stored as `sha256` hashes in `user_sessions`

**JWT claims** (see `pkg/token/jwt.go`): `UserID`, `Email`, `Roles []string`

**Adding a new domain:**
1. Create `internal/<domain>/` with `model.go`, `repository.go`, `service.go`, `handler.go`
2. Wire it in `internal/server/server.go` `routes()` method
3. Add migrations under `backend/migrations/`

### Frontend (`frontend/`)

**Key config:**
- `svelte.config.js` — uses `@sveltejs/adapter-node` (port 3000); SSR disabled globally via `src/routes/+layout.ts` (`export const ssr = false`)
- Vite proxies `/api` to the backend at `http://localhost:8080` during dev (check `vite.config.ts`)
- Svelte 5 runes (`$state`, `$props`, `$derived`) — **not** the legacy options API

**Path aliases (use these everywhere):**
| Alias | Resolves to |
|---|---|
| `$components` | `src/lib/components` |
| `$stores` | `src/lib/stores` |
| `$api` | `src/lib/api` |
| `$utils` | `src/lib/utils` |
| `$types` | `src/lib/types` |

**Route groups:**
- `(auth)/` — unauthenticated pages (login, register, forgot-password)
- `(app)/` — authenticated pages with sidebar layout; redirects to `/login` if no user

**State management:**
- `$stores/auth` — `authStore` (user, accessToken, loading), derived stores: `isAuthenticated`, `currentUser`, `isLoading`
- `$stores/toast` — `toast.info/success/warning/error(title, body)`

**API layer:**
- `$api/client.ts` — `ApiClient` class; access token held in memory; `credentials: 'include'` for cookie-based refresh; unwraps `response.data` automatically
- `$api/auth.ts` — typed wrappers for auth endpoints

**Styling:**
- TailwindCSS v4 with CSS custom properties for theming (e.g., `var(--color-primary)`, `var(--color-background)`, `var(--color-border)`)
- Icon library: `lucide-svelte`

## Environment Variables

Required (will panic at startup if missing): `DATABASE_URL`, `REDIS_URL`, `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`

Copy `backend/.env.example` → `backend/.env` and `frontend/.env.example` → `frontend/.env` (`make setup` does this automatically).

## Database

- PostgreSQL via `pgx/v5` (direct queries, no ORM)
- Migrations: numbered `NNNNNN_description.up.sql` / `.down.sql` in `backend/migrations/`
- Seed data is part of migration `000012` (roles & permissions); run `make migrate` to apply

## Go Module

`github.com/404nfid/go-svelte-starter-kit` — use this import path for all internal packages.
