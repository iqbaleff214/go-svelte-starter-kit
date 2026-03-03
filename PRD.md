# Product Requirements Document (PRD)
## Go + Svelte Full-Stack Starter Kit

**Version:** 1.0.0
**Date:** 2026-03-03
**Status:** Draft

---

## 1. Overview

### 1.1 Product Summary

A production-ready, opinionated full-stack starter kit that combines **SvelteKit** (frontend) with **Go** (backend) to provide developers with a solid foundation for building modern web applications. The kit ships with a comprehensive set of pre-built features — authentication, real-time notifications, RBAC, AI integration, and a public API layer — so teams can focus on product logic instead of boilerplate.

### 1.2 Goals

- Eliminate weeks of initial setup work for common application requirements.
- Provide a clean, extensible architecture that scales from MVP to production.
- Deliver a polished, mobile-first UI out of the box.
- Be opinionated enough to guide best practices, but flexible enough to customize.

### 1.3 Non-Goals

- This is not a SaaS product — it is a code template/starter.
- Does not include payment processing, multi-tenancy, or CMS features (out of scope for v1).
- Not intended for monolith deployments only — should support containerized/cloud-native deployment.

---

## 2. Tech Stack

| Layer        | Technology                                      |
|--------------|-------------------------------------------------|
| Frontend     | SvelteKit 2.x, TypeScript, TailwindCSS          |
| Backend      | Go 1.22+, Chi or Fiber router                   |
| Database     | PostgreSQL (primary), Redis (cache & queues)    |
| Auth         | JWT (access + refresh tokens), Paseto optional  |
| Email        | SMTP / SendGrid via background worker           |
| Real-time    | WebSocket (Gorilla WS or nhooyr/websocket)      |
| AI           | Anthropic Claude API (claude-sonnet-4-6)        |
| Queue/Worker | Redis-based job queue (asynq or river)          |
| ORM/DB       | sqlc or GORM                                    |
| Containerize | Docker + Docker Compose                         |

---

## 3. Feature Requirements

---

### 3.1 Authentication

**Description:** Core authentication system supporting email/password, social login, and token management.

**User Stories:**
- As a user, I can register with my email and password.
- As a user, I can log in and receive a JWT access token and refresh token.
- As a user, I can log out and have my tokens invalidated.
- As a developer, tokens are automatically refreshed on the frontend before expiry.

**Acceptance Criteria:**
- [ ] Email/password registration with email verification.
- [ ] Login endpoint returns short-lived access token (15 min) + long-lived refresh token (7 days).
- [ ] Refresh token rotation implemented (old token invalidated on use).
- [ ] Tokens stored securely: access token in memory, refresh token in `HttpOnly` cookie.
- [ ] Rate limiting on auth endpoints (e.g., 5 attempts/min per IP).
- [ ] Password hashing with bcrypt (cost ≥ 12).
- [ ] Account lockout after repeated failed logins (configurable threshold).
- [ ] Secure password reset flow via email token (expires in 1 hour).

---

### 3.2 Social Login (Google OAuth)

**Description:** Allow users to sign in with their Google account.

**User Stories:**
- As a user, I can click "Sign in with Google" and authenticate without creating a password.
- As a user, if I already have an email/password account, signing in with Google links both.

**Acceptance Criteria:**
- [ ] Google OAuth2 PKCE flow implemented.
- [ ] Backend handles OAuth callback, creates or links user account.
- [ ] User profile photo and display name synced from Google on first login.
- [ ] Extensible OAuth provider interface — adding GitHub/LinkedIn is a matter of implementing one interface.
- [ ] State parameter used to prevent CSRF attacks.

---

### 3.3 Two-Factor Authentication (2FA)

**Description:** Optional TOTP-based 2FA for enhanced account security.

**User Stories:**
- As a user, I can enable 2FA using an authenticator app (Google Authenticator, Authy).
- As a user, when 2FA is enabled, I must provide a 6-digit code after my password.
- As a user, I can generate and save backup codes in case I lose access to my authenticator.

**Acceptance Criteria:**
- [ ] TOTP generation using `pquerna/otp` or equivalent Go library.
- [ ] QR code rendered on setup page for scanning with authenticator app.
- [ ] 10 single-use backup codes generated on 2FA activation.
- [ ] 2FA enforcement can be made mandatory by role/admin.
- [ ] 2FA status visible in security settings.
- [ ] Disabling 2FA requires current TOTP code or backup code.

---

### 3.4 Profile Management

**Description:** Authenticated users can view and edit their personal profile.

**User Stories:**
- As a user, I can update my display name, bio, avatar, and contact info.
- As a user, I can change my password from the profile page.
- As a user, I can view my active sessions and revoke them.
- As a user, I can delete my account (soft delete with 30-day grace period).

**Acceptance Criteria:**
- [ ] Profile avatar upload with image resizing (max 2MB, WebP/JPEG/PNG).
- [ ] Validation on all fields (name length, bio character limit, etc.).
- [ ] Email change requires re-verification of new email.
- [ ] Active sessions list shows device, IP, last seen, with revoke option.
- [ ] Account deletion sends confirmation email and schedules background cleanup.

---

### 3.5 Email System with Background Worker

**Description:** Reliable, async email delivery via a background worker queue.

**User Stories:**
- As a developer, I can enqueue emails from anywhere in the codebase with a simple API.
- As an operator, failed emails are retried with exponential backoff and logged.

**Acceptance Criteria:**
- [ ] Email worker powered by Redis queue (asynq or river).
- [ ] Pre-built email templates for: welcome, email verification, password reset, 2FA backup codes, security alerts.
- [ ] HTML email templates using Go `html/template` with a base layout.
- [ ] SMTP and SendGrid adapters included; switchable via config.
- [ ] Dead-letter queue for permanently failed emails with admin visibility.
- [ ] Rate limiting on outbound email per user (prevent spam abuse).
- [ ] Email delivery logs stored in DB (status, timestamps, error reason).

**Email Templates:**
| Template              | Trigger                        |
|-----------------------|--------------------------------|
| Welcome               | On registration                |
| Email Verification    | On registration / email change |
| Password Reset        | On reset request               |
| 2FA Backup Codes      | On 2FA activation              |
| Security Alert        | Login from new device/IP       |
| Account Deletion      | On deletion request            |
| Notification Digest   | Scheduled (optional)           |

---

### 3.6 Real-Time Notification System (WebSocket)

**Description:** Push real-time notifications to connected users via WebSocket.

**User Stories:**
- As a user, I receive real-time notifications without refreshing the page.
- As a user, I can view my notification history and mark them as read.
- As a developer, I can publish a notification to a user from any service layer.

**Acceptance Criteria:**
- [ ] WebSocket hub manages per-user connections (handles multiple tabs/devices).
- [ ] Notifications persisted to DB before delivery (no loss on disconnect).
- [ ] Unread notification count badge in the UI, updated in real-time.
- [ ] Notification types: `info`, `success`, `warning`, `alert`.
- [ ] Notification payload: `id`, `type`, `title`, `body`, `link` (optional), `read_at`, `created_at`.
- [ ] Pagination on notification history endpoint.
- [ ] Mark single / mark all as read endpoints.
- [ ] Heartbeat/ping-pong to detect stale connections.
- [ ] Graceful reconnect logic on the frontend (exponential backoff).

---

### 3.7 Role-Based Access Control (RBAC)

**Description:** Flexible role and permission system to control resource access.

**User Stories:**
- As an admin, I can assign roles to users and configure what each role can do.
- As a developer, I can protect routes and actions with permission guards.

**Acceptance Criteria:**
- [ ] Pre-seeded roles: `superadmin`, `admin`, `user`.
- [ ] Permissions are granular strings: e.g., `users:read`, `users:write`, `admin:access`.
- [ ] Middleware guard on backend routes: `RequirePermission("users:write")`.
- [ ] Role-to-permission mapping stored in DB (not hardcoded).
- [ ] Frontend route guards check user roles from JWT claims.
- [ ] Admin panel to manage roles and assign permissions.
- [ ] Users can have multiple roles (many-to-many).
- [ ] Permission checks cached in Redis with TTL to avoid repeated DB hits.

**Default Role Matrix:**

| Permission          | Superadmin | Admin | User |
|---------------------|:----------:|:-----:|:----:|
| `users:read`        | ✓          | ✓     | -    |
| `users:write`       | ✓          | ✓     | -    |
| `users:delete`      | ✓          | -     | -    |
| `roles:manage`      | ✓          | -     | -    |
| `notifications:*`   | ✓          | ✓     | ✓    |
| `profile:*`         | ✓          | ✓     | ✓    |
| `api_keys:manage`   | ✓          | ✓     | -    |

---

### 3.8 AI Agent Integration

**Description:** Built-in integration with Anthropic's Claude API, exposing a chat/agent endpoint that frontend can consume.

**User Stories:**
- As a user, I can interact with an AI assistant embedded in the application.
- As a developer, I can extend the AI agent with custom tools and system prompts.

**Acceptance Criteria:**
- [ ] Backend proxy endpoint for Claude API (`/api/ai/chat`) with auth protection.
- [ ] Streaming response support (Server-Sent Events or streamed WebSocket).
- [ ] Conversation history stored per user session (persisted in DB, purged after configurable TTL).
- [ ] Tool/function calling scaffolding — define tools in Go, expose them to the agent.
- [ ] System prompt configurable per use case via config/env.
- [ ] Rate limiting per user on AI endpoints.
- [ ] Token usage tracked and logged per request.
- [ ] Frontend chat UI component included (message list, input, streaming indicator).
- [ ] Default model: `claude-sonnet-4-6`; configurable via environment variable.

**Pre-built Agent Tools:**
| Tool                 | Description                          |
|----------------------|--------------------------------------|
| `get_current_user`   | Returns the current user's profile   |
| `list_notifications` | Lists user's recent notifications    |
| `search_users`       | Admin-only user search               |

---

### 3.9 Public API

**Description:** A versioned, documented REST API for third-party integrations with API key authentication.

**User Stories:**
- As a developer, I can generate an API key from my profile.
- As an external system, I can authenticate with an API key and access permitted resources.
- As a developer, I can explore the API via auto-generated documentation.

**Acceptance Criteria:**
- [ ] API versioning: `/api/v1/`.
- [ ] API key generation and management (create, list, revoke) per user.
- [ ] API keys hashed in DB; only shown once at creation.
- [ ] API key scopes/permissions aligned with RBAC (e.g., `read:profile`).
- [ ] Rate limiting per API key (configurable: req/min, req/day).
- [ ] OpenAPI 3.0 spec auto-generated or maintained.
- [ ] Swagger UI served at `/api/docs` in dev/staging.
- [ ] Audit log of all API key usage (endpoint, timestamp, status code).
- [ ] API key expiry (optional, configurable at creation).

**Public API Endpoints (v1):**

| Method | Endpoint                    | Description                   | Auth       |
|--------|-----------------------------|-------------------------------|------------|
| GET    | `/api/v1/me`                | Get current user profile      | API Key    |
| PATCH  | `/api/v1/me`                | Update profile                | API Key    |
| GET    | `/api/v1/notifications`     | List notifications            | API Key    |
| PATCH  | `/api/v1/notifications/:id` | Mark notification read        | API Key    |
| GET    | `/api/v1/users`             | List users (admin)            | API Key    |
| GET    | `/api/v1/users/:id`         | Get user by ID (admin)        | API Key    |
| POST   | `/api/v1/webhooks`          | Register a webhook endpoint   | API Key    |
| DELETE | `/api/v1/webhooks/:id`      | Remove webhook                | API Key    |

---

## 4. Frontend Requirements

### 4.1 Design System

- **CSS Framework:** TailwindCSS v4 with custom design tokens.
- **Component Library:** Custom components (no heavy third-party UI lib dependency).
- **Icon Set:** Lucide Icons or Heroicons.
- **Typography:** Inter (system fallback stack).
- **Color:** Neutral base palette with configurable primary accent color via CSS variables.
- **Dark Mode:** Full dark/light mode support with system preference detection.

### 4.2 Mobile-First & Responsive

- All layouts built mobile-first (320px baseline).
- Sidebar navigation collapses to bottom tab bar on mobile.
- Touch-friendly tap targets (min 44×44px).
- No horizontal scroll on any viewport.
- Tested at: 375px (iPhone SE), 768px (iPad), 1280px (Desktop), 1920px (Wide).

### 4.3 Pages & Routes

| Route                    | Description                        | Auth Required |
|--------------------------|------------------------------------|:-------------:|
| `/`                      | Landing / marketing page           | No            |
| `/login`                 | Login (email + social)             | No            |
| `/register`              | Registration                       | No            |
| `/forgot-password`       | Password reset request             | No            |
| `/reset-password`        | Password reset form (token)        | No            |
| `/verify-email`          | Email verification                 | No            |
| `/dashboard`             | Main app dashboard                 | Yes           |
| `/profile`               | User profile & settings            | Yes           |
| `/profile/security`      | 2FA, sessions, password change     | Yes           |
| `/profile/api-keys`      | API key management                 | Yes           |
| `/notifications`         | Notification history               | Yes           |
| `/ai`                    | AI assistant chat                  | Yes           |
| `/admin`                 | Admin overview                     | Admin         |
| `/admin/users`           | User management                    | Admin         |
| `/admin/roles`           | Role & permission management       | Admin         |
| `/admin/emails`          | Email delivery logs                | Admin         |

### 4.4 UI Components

- Auth forms (login, register, reset password) with inline validation.
- Toast notification component (top-right, auto-dismiss).
- Real-time notification bell with dropdown.
- User avatar with fallback initials.
- Data tables with sorting, filtering, pagination.
- Confirmation dialog (modal).
- Loading skeletons for async data.
- Chat interface with streaming message support.
- QR code display for 2FA setup.
- Code display for backup codes.
- API key reveal/copy widget.

---

## 5. Backend Requirements

### 5.1 Project Structure

```
/
├── cmd/
│   ├── api/          # HTTP server entrypoint
│   └── worker/       # Background worker entrypoint
├── internal/
│   ├── auth/         # Auth logic, JWT, OAuth
│   ├── user/         # User domain
│   ├── notification/ # Notification domain
│   ├── email/        # Email templates & dispatch
│   ├── ai/           # AI agent integration
│   ├── rbac/         # Role/permission management
│   ├── api/          # Public API handlers
│   ├── ws/           # WebSocket hub
│   └── middleware/   # Auth, rate limit, RBAC middleware
├── pkg/
│   ├── config/       # Env/config loading
│   ├── database/     # DB connection, migrations
│   ├── redis/        # Redis client
│   ├── logger/       # Structured logging
│   └── validator/    # Request validation
├── migrations/       # SQL migration files
├── templates/        # Email HTML templates
└── docker-compose.yml
```

### 5.2 API Design Principles

- RESTful conventions for resource endpoints.
- Consistent error response envelope: `{ "error": { "code": "...", "message": "...", "details": {...} } }`.
- Consistent success response envelope: `{ "data": {...}, "meta": {...} }`.
- Pagination via cursor or offset with `limit`, `offset`/`cursor`, `total` in meta.
- All timestamps in ISO 8601 / RFC 3339 UTC format.
- Input validation with structured error messages per field.

### 5.3 Security Requirements

- All API endpoints served over HTTPS (TLS termination at reverse proxy).
- CORS configured to allowlist specific origins.
- `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options` headers set.
- SQL injection prevented via parameterized queries (sqlc/GORM).
- Input sanitization on all user-supplied fields.
- Secrets managed via environment variables (no hardcoded secrets).
- `.env.example` provided; `.env` gitignored.
- Dependency audit in CI pipeline.

### 5.4 Observability

- Structured JSON logging (zerolog or slog).
- Request/response logging middleware (exclude sensitive fields).
- Health check endpoint: `GET /health`.
- Readiness check endpoint: `GET /ready` (checks DB + Redis).
- Metrics endpoint (optional): Prometheus-compatible `/metrics`.

---

## 6. Database Schema (High-Level)

```
users
  id, email, password_hash, display_name, avatar_url, bio,
  email_verified_at, two_fa_enabled, two_fa_secret,
  deleted_at, created_at, updated_at

oauth_providers
  id, user_id, provider (google|github), provider_user_id,
  access_token, refresh_token, expires_at, created_at

user_sessions
  id, user_id, refresh_token_hash, user_agent, ip_address,
  last_seen_at, expires_at, revoked_at, created_at

two_fa_backup_codes
  id, user_id, code_hash, used_at, created_at

roles
  id, name, description, created_at, updated_at

permissions
  id, name, description, created_at

role_permissions
  role_id, permission_id

user_roles
  user_id, role_id, assigned_by, assigned_at

notifications
  id, user_id, type, title, body, link, read_at, created_at

email_logs
  id, user_id, template, recipient, status, error, sent_at, created_at

api_keys
  id, user_id, name, key_hash, key_prefix, scopes, last_used_at,
  expires_at, revoked_at, created_at

api_key_logs
  id, api_key_id, method, path, status_code, ip, created_at

ai_conversations
  id, user_id, messages (jsonb), model, token_usage, created_at, updated_at

webhooks
  id, user_id, url, events, secret_hash, active, created_at, updated_at
```

---

## 7. Configuration & Environment

```env
# App
APP_ENV=development
APP_PORT=8080
APP_URL=http://localhost:8080
FRONTEND_URL=http://localhost:5173

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/starter_kit

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_ACCESS_SECRET=...
JWT_REFRESH_SECRET=...

# Google OAuth
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback

# Email
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=...
SMTP_PASS=...
SMTP_FROM=noreply@example.com
# OR
SENDGRID_API_KEY=...

# Anthropic AI
ANTHROPIC_API_KEY=...
ANTHROPIC_MODEL=claude-sonnet-4-6

# Rate Limiting
RATE_LIMIT_AUTH=5
RATE_LIMIT_API=100
```

---

## 8. Developer Experience

### 8.1 Getting Started

- Single command setup: `docker compose up` brings up Postgres, Redis, API, Worker, and Frontend.
- `make dev` runs hot-reload for both Go backend and SvelteKit frontend.
- `make migrate` runs database migrations.
- `make seed` populates development data.
- `make test` runs all tests.

### 8.2 Documentation

- `README.md` with quick start, architecture diagram, and feature overview.
- Each feature domain has its own `README.md` under `internal/<domain>/`.
- OpenAPI spec at `/api/docs`.
- Inline code comments on non-obvious logic.

### 8.3 Testing

- Backend unit tests for business logic (auth, RBAC, notifications).
- Integration tests for HTTP handlers using `httptest`.
- Frontend component tests with Vitest + Testing Library.
- E2E tests with Playwright for critical flows (login, 2FA, profile update).
- Test coverage target: ≥ 70% on backend business logic.

---

## 9. Deployment

### 9.1 Docker

- Multi-stage Dockerfile for Go backend (final image: `gcr.io/distroless/base`).
- SvelteKit built to static/Node adapter.
- `docker-compose.yml` for local development.
- `docker-compose.prod.yml` for production reference.

### 9.2 CI/CD

- GitHub Actions workflows:
  - `ci.yml`: lint, test, build on every PR.
  - `release.yml`: build and push Docker images on tag.
- Linting: `golangci-lint` for Go, `eslint` + `prettier` for TS/Svelte.

### 9.3 Reverse Proxy Reference

- Nginx or Caddy configuration example provided.
- TLS termination at proxy layer.
- WebSocket upgrade headers configured.

---

## 10. Milestones & Phases

| Phase | Features                                                              | Target  |
|-------|-----------------------------------------------------------------------|---------|
| 1     | Project scaffold, DB schema, Auth (email/pass), JWT, basic UI        | Week 1-2 |
| 2     | Google OAuth, 2FA, Profile management, session management            | Week 3-4 |
| 3     | Email worker system, all email templates, email logs                  | Week 5   |
| 4     | WebSocket notification system, notification UI                        | Week 6   |
| 5     | RBAC system, admin panel (users, roles, permissions)                 | Week 7-8 |
| 6     | AI agent integration, chat UI, streaming, tool calling               | Week 9   |
| 7     | Public API, API key management, OpenAPI docs, audit logs             | Week 10  |
| 8     | E2E tests, Playwright, polish, mobile QA, documentation              | Week 11-12 |

---

## 11. Success Metrics

- Developer can clone and have a running app in under 5 minutes.
- All listed features functional and tested.
- Lighthouse score ≥ 90 (Performance, Accessibility, Best Practices) on mobile.
- Zero known critical/high security vulnerabilities at release.
- Core backend test coverage ≥ 70%.
- README rated clear by at least 3 external developers during review.

---

## 12. Open Questions

| # | Question                                              | Owner     | Status  |
|---|-------------------------------------------------------|-----------|---------|
| 1 | Use `sqlc` or `GORM` as DB layer?                     | Tech Lead | Open    |
| 2 | Should AI chat history be end-to-end encrypted?       | Product   | Open    |
| 3 | Include GitHub OAuth in v1 or defer to v2?            | Product   | Open    |
| 4 | Webhook delivery system scope for v1?                 | Tech Lead | Open    |
| 5 | Should admin panel be a separate SvelteKit app/route? | Tech Lead | Open    |
| 6 | SvelteKit adapter: Node, static, or Cloudflare?       | Infra     | Open    |

---

*This document is a living artifact. Update as decisions are made and scope is refined.*
