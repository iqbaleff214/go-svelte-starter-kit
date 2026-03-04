# Go + SvelteKit Starter Kit

![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![SvelteKit](https://img.shields.io/badge/SvelteKit-2.x-FF3E00?logo=svelte&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

A production-ready full-stack starter kit pairing a **Go** API backend with a **SvelteKit** CSR frontend — batteries included for auth, RBAC, notifications, AI chat, and a public API.

---

## Features

- **Auth** — Email/password + Google OAuth, JWT (access + refresh rotation), 2FA (TOTP)
- **RBAC** — Role-based access control (`superadmin`, `admin`, `user`) with permission guards
- **Notifications** — Real-time push via WebSocket + persistent notification history
- **AI Chat** — Streaming chat completions via Anthropic Claude
- **Public API** — API key management (`sk_*` prefix), per-key scopes, rate limiting, audit logs
- **Webhooks** — Register URLs to receive event callbacks
- **Admin Panel** — User management, role assignment, system stats
- **Email** — Transactional email (SMTP or SendGrid) for verification, password reset

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22, [Chi](https://github.com/go-chi/chi) router |
| Frontend | SvelteKit 2.x + Svelte 5 (runes), TypeScript |
| Database | PostgreSQL (pgx/v5 — no ORM) |
| Cache / PubSub | Redis |
| Auth | JWT (golang-jwt), bcrypt, TOTP (pquerna/otp) |
| AI | Anthropic Claude API (streaming) |
| Email | SMTP / SendGrid |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Styling | TailwindCSS v4 |
| Icons | lucide-svelte |
| Containerisation | Docker + Docker Compose |

---

## Quick Start

```sh
# 1. Copy env files and install dependencies
make setup

# 2. Start Postgres + Redis
make infra

# 3. Apply database migrations
make migrate

# 4. Start backend + frontend with hot-reload
make dev
```

- Backend API: http://localhost:8080
- Frontend: http://localhost:5173
- API Docs (Swagger UI): http://localhost:8080/api/docs

---

## Architecture

```
.
├── backend/                  Go API server + background worker
│   ├── cmd/api/              Entrypoint (HTTP server)
│   ├── cmd/worker/           Entrypoint (background jobs)
│   ├── internal/             Feature domains (auth, user, notification, …)
│   │   ├── <domain>/
│   │   │   ├── handler.go    HTTP handlers (Chi)
│   │   │   ├── service.go    Business logic
│   │   │   ├── repository.go Database queries (pgx)
│   │   │   └── model.go      Types + request/response structs
│   │   ├── middleware/       Auth, RBAC, rate-limit, logging
│   │   └── server/           Router wiring + embedded OpenAPI spec
│   ├── migrations/           Sequential SQL migrations (NNNNNN_name.up/down.sql)
│   └── pkg/                  Shared infrastructure (config, db, redis, token, …)
├── frontend/                 SvelteKit app (CSR-only, adapter-node)
│   ├── src/lib/
│   │   ├── api/              Typed fetch wrappers ($api alias)
│   │   ├── components/       Reusable UI components ($components alias)
│   │   ├── stores/           Svelte stores — auth, toast, notifications ($stores alias)
│   │   └── types/            Shared TypeScript interfaces ($types alias)
│   ├── src/routes/
│   │   ├── (auth)/           Public pages: login, register, forgot-password
│   │   └── (app)/            Authenticated pages with sidebar layout
│   └── e2e/                  Playwright end-to-end tests
├── docker-compose.yml        Production compose
├── docker-compose.dev.yml    Dev infrastructure (Postgres + Redis only)
└── Makefile                  Unified dev commands
```

---

## Environment Variables

Copy `backend/.env.example` → `backend/.env` (done automatically by `make setup`).

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `JWT_ACCESS_SECRET` | Yes | Secret for signing access tokens (≥32 chars) |
| `JWT_REFRESH_SECRET` | Yes | Secret for signing refresh tokens (≥32 chars) |
| `APP_ENV` | No | `development` or `production` (default: `development`) |
| `APP_PORT` | No | HTTP listen port (default: `8080`) |
| `GOOGLE_CLIENT_ID` | No | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | No | Google OAuth client secret |
| `EMAIL_PROVIDER` | No | `smtp` or `sendgrid` |
| `SMTP_HOST` / `SMTP_USER` / `SMTP_PASS` | No | SMTP credentials |
| `SENDGRID_API_KEY` | No | SendGrid API key (if using sendgrid provider) |
| `ANTHROPIC_API_KEY` | No | Anthropic API key for AI chat |
| `RATE_LIMIT_AUTH` | No | Auth endpoint rate limit req/min (default: `5`) |
| `RATE_LIMIT_API` | No | General API rate limit req/min (default: `100`) |

---

## Available Commands

| Command | Description |
|---|---|
| `make setup` | First-time setup: copy env files, install deps |
| `make dev` | Start infra + backend + frontend (hot-reload) |
| `make infra` | Start Postgres + Redis only |
| `make stop` | Stop all dev infrastructure |
| `make migrate` | Apply all pending migrations |
| `make migrate-down` | Roll back the last migration |
| `make migrate-create MIGRATION_NAME=name` | Create a new migration pair |
| `make migrate-status` | Show current migration version |
| `make test` | Run all backend tests |
| `make test-cover` | Run backend tests with HTML coverage report |
| `make test-fe` | Run frontend unit tests (Vitest) |
| `make lint` | Lint backend (golangci-lint) + frontend (ESLint) |
| `make fmt` | Format backend (gofmt) + frontend (Prettier) |
| `make build-be` | Build backend binaries → `backend/bin/` |
| `make build-fe` | Build frontend for production |
| `make build` | Build production Docker images |

---

## Tests

**Backend unit tests** (no database required):
```sh
make test
# or run a single test:
cd backend && go test ./pkg/token/... -run TestGeneratePair -v
```

**Frontend unit tests** (Vitest):
```sh
make test-fe
```

**End-to-end tests** (Playwright — requires running dev stack):
```sh
make dev          # in one terminal
cd frontend && npm run test:e2e   # in another
```

For E2E tests that require a real user account, set `TEST_EMAIL` and `TEST_PASSWORD` in your environment.

---

## Deployment

A `docker-compose.yml` is provided for production use:

```sh
docker compose up -d
```

The compose file expects the same environment variables as `.env.example`. Mount secrets via Docker secrets or an external env file.

---

## License

MIT
