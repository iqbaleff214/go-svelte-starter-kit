# Go + SvelteKit Starter Kit

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/SvelteKit-2.x-FF3E00?style=flat-square&logo=svelte&logoColor=white" alt="SvelteKit" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/License-MIT-22c55e?style=flat-square" alt="License" />
</p>

A production-ready full-stack starter kit pairing a **Go** API backend with a **SvelteKit** CSR frontend. Ships with authentication, role-based access control, real-time notifications, AI chat, a public API layer, and full Docker support — ready to clone and build on.

> **Author:** M. Iqbal Effendi — [iqbaleff214@gmail.com](mailto:iqbaleff214@gmail.com)

---

## Features

| Feature | Description |
|---|---|
| **Authentication** | Email/password, Google OAuth 2.0 (PKCE), JWT access + refresh token rotation, 2FA via TOTP |
| **RBAC** | Role-based access control with roles (`superadmin`, `admin`, `user`) and granular permission guards |
| **Real-time Notifications** | WebSocket-powered push notifications with persistent history |
| **AI Chat** | Streaming chat via [OpenRouter](https://openrouter.ai) (OpenAI-compatible), tool calling, conversation history |
| **Public API** | Scoped API key management (`sk_*` prefix), per-key rate limiting, audit logs |
| **Webhooks** | Register and receive event-driven HTTP callbacks |
| **Admin Panel** | User management, role assignment, system statistics |
| **Transactional Email** | SMTP or SendGrid for email verification and password reset flows |

---

## Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.22+, [Chi](https://github.com/go-chi/chi) router |
| **Frontend** | SvelteKit 2.x + Svelte 5 (runes), TypeScript |
| **Database** | PostgreSQL via [pgx/v5](https://github.com/jackc/pgx) — no ORM |
| **Cache / Queue** | Redis |
| **Auth** | JWT ([golang-jwt](https://github.com/golang-jwt/jwt)), bcrypt, TOTP ([pquerna/otp](https://github.com/pquerna/otp)) |
| **AI** | [OpenRouter](https://openrouter.ai) — streaming, OpenAI-compatible |
| **Email** | SMTP / SendGrid |
| **Migrations** | [golang-migrate](https://github.com/golang-migrate/migrate) |
| **Styling** | TailwindCSS v4 |
| **Icons** | [lucide-svelte](https://lucide.dev) |
| **Containerization** | Docker + Docker Compose |

---

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose

### Quick Start

```sh
# 1. Clone the repository
git clone https://github.com/404nfid/go-svelte-starter-kit.git
cd go-svelte-starter-kit

# 2. Copy env files and install dependencies
make setup

# 3. Start PostgreSQL + Redis
make infra

# 4. Apply database migrations
make migrate

# 5. Start backend + frontend with hot-reload
make dev
```

| Service | URL |
|---|---|
| Frontend (Vite dev server) | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| API Documentation (Swagger) | http://localhost:8080/api/docs |

---

## Project Structure

```
.
├── backend/                        Go API server + background worker
│   ├── cmd/
│   │   ├── api/                    HTTP server entrypoint
│   │   └── worker/                 Background job worker entrypoint
│   ├── internal/                   Feature domains
│   │   ├── <domain>/
│   │   │   ├── handler.go          HTTP handlers (Chi)
│   │   │   ├── service.go          Business logic
│   │   │   ├── repository.go       Database queries (pgx)
│   │   │   └── model.go            Types and request/response structs
│   │   ├── middleware/             Auth, RBAC, rate limiting, request logging
│   │   └── server/                 Router wiring, middleware setup
│   ├── migrations/                 Sequential SQL migrations (NNNNNN_name.up/down.sql)
│   └── pkg/                        Shared infrastructure (config, db, redis, token, logger)
│
└── frontend/                       SvelteKit application (CSR-only, adapter-node)
    └── src/
        ├── lib/
        │   ├── api/                Typed fetch wrappers  ($api alias)
        │   ├── components/         Reusable UI components ($components alias)
        │   ├── stores/             Svelte stores — auth, toast, theme ($stores alias)
        │   └── types/              Shared TypeScript interfaces ($types alias)
        └── routes/
            ├── (auth)/             Public pages: login, register, forgot/reset password
            └── (app)/              Authenticated pages with sidebar shell layout
```

---

## Configuration

Copy `backend/.env.example` → `backend/.env` (`make setup` does this automatically).

### Required Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_ACCESS_SECRET` | Secret for signing access tokens (minimum 32 characters) |
| `JWT_REFRESH_SECRET` | Secret for signing refresh tokens (minimum 32 characters) |

### Optional Variables

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Environment — `development` or `production` |
| `APP_PORT` | `8080` | HTTP listen port |
| `GOOGLE_CLIENT_ID` | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret |
| `EMAIL_PROVIDER` | `smtp` | Email backend — `smtp` or `sendgrid` |
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASS` | — | SMTP password |
| `SENDGRID_API_KEY` | — | SendGrid API key |
| `OPENROUTER_API_KEY` | — | OpenRouter API key for AI chat |
| `OPENROUTER_MODEL` | `openrouter/free` | Model identifier (any OpenRouter-compatible model slug) |
| `RATE_LIMIT_AUTH` | `5` | Auth endpoint rate limit (requests/min) |
| `RATE_LIMIT_API` | `100` | General API rate limit (requests/min) |

---

## Available Commands

### Development

| Command | Description |
|---|---|
| `make setup` | First-time setup: copy env files, install frontend deps, download Go modules |
| `make dev` | Start infrastructure + backend + frontend with hot-reload |
| `make infra` | Start PostgreSQL and Redis only |
| `make stop` | Stop all development infrastructure |

### Database

| Command | Description |
|---|---|
| `make migrate` | Apply all pending migrations |
| `make migrate-down` | Roll back the last migration |
| `make migrate-create MIGRATION_NAME=name` | Scaffold a new migration pair |
| `make migrate-status` | Show current migration version |

### Quality

| Command | Description |
|---|---|
| `make test` | Run all backend tests |
| `make test-cover` | Run backend tests with HTML coverage report |
| `make test-fe` | Run frontend unit tests (Vitest) |
| `make lint` | Lint backend (golangci-lint) + frontend (ESLint) |
| `make fmt` | Format backend (gofmt) + frontend (Prettier) |

### Build

| Command | Description |
|---|---|
| `make build-be` | Build backend binaries → `backend/bin/` |
| `make build-fe` | Build frontend for production |
| `make build` | Build production Docker images |

---

## Testing

**Backend** (unit tests, no database required):
```sh
make test

# Run a single test suite:
cd backend && go test ./internal/auth/... -run TestServiceLogin -v
```

**Frontend** (Vitest):
```sh
make test-fe
```

**End-to-end** (Playwright — requires running dev stack):
```sh
# Terminal 1
make dev

# Terminal 2
cd frontend && npm run test:e2e
```

> For E2E tests that require an authenticated user, set `TEST_EMAIL` and `TEST_PASSWORD` in your environment before running.

---

## Deployment

A production-ready `docker-compose.yml` is included:

```sh
docker compose up -d
```

The compose file reads from the same environment variables defined in `.env.example`. For production, supply secrets via Docker secrets, a `.env` file mounted at runtime, or your platform's secret management system.

---

## License

This project is licensed under the [MIT License](LICENSE).
