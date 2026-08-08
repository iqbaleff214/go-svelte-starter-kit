# Deployment Guide

This document covers every supported deployment method for the Go + SvelteKit Starter Kit — from a single Docker Compose command to fully manual bare-metal setups and cloud platforms.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Configuration](#environment-configuration)
3. [Database Migrations](#database-migrations)
4. [Method 1 — Docker Compose (Recommended)](#method-1--docker-compose-recommended)
5. [Method 2 — Manual Bare-Metal](#method-2--manual-bare-metal)
6. [Method 3 — Manual with systemd (Linux)](#method-3--manual-with-systemd-linux)
7. [Method 4 — Reverse Proxy with Nginx (no Docker)](#method-4--reverse-proxy-with-nginx-no-docker)
8. [Method 5 — Fly.io](#method-5--flyio)
9. [Method 6 — Railway](#method-6--railway)
10. [Method 7 — Render](#method-7--render)
11. [SSL / TLS (HTTPS)](#ssl--tls-https)
12. [Health Checks & Monitoring](#health-checks--monitoring)
13. [Upgrading](#upgrading)

---

## Prerequisites

| Tool | Version | Required for |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.22+ | Building backend (manual deploys) |
| [Node.js](https://nodejs.org/) | 20+ | Building frontend (manual deploys) |
| [Docker](https://docs.docker.com/get-docker/) | 24+ | Docker-based deploys |
| [Docker Compose](https://docs.docker.com/compose/) | v2 | Docker-based deploys |
| PostgreSQL | 16+ | All methods (external or containerized) |
| Redis | 7+ | All methods (external or containerized) |

---

## Environment Configuration

All backend configuration is read from environment variables. Copy the example file before any deployment:

```sh
cp backend/.env.example backend/.env
```

Then edit `backend/.env`. At minimum, set all **required** variables:

```env
# Required
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=require
REDIS_URL=redis://host:6379
JWT_ACCESS_SECRET=<random-string-min-32-chars>
JWT_REFRESH_SECRET=<random-string-min-32-chars>

# Recommended for production
APP_ENV=production
APP_URL=https://yourdomain.com
FRONTEND_URL=https://yourdomain.com
```

Generate strong secrets with:

```sh
openssl rand -hex 32
```

> **Never commit `backend/.env` to version control.** It is already in `.gitignore`.

---

## Database Migrations

Migrations must be run once after the database is ready, and again after each upgrade. They are applied using the `migrate/migrate` Docker image:

```sh
# Apply all pending migrations
make migrate

# Or with a custom DATABASE_URL
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require make migrate
```

To run migrations without Docker (using the `migrate` CLI binary directly):

```sh
# Install the CLI: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
migrate -path backend/migrations -database "$DATABASE_URL" up
```

---

## Method 1 — Docker Compose (Recommended)

The simplest production deployment. Starts PostgreSQL, Redis, the Go API, the background worker, the SvelteKit frontend, and an nginx reverse proxy in a single command.

### 1. Prepare environment

```sh
cp backend/.env.example backend/.env
# Edit backend/.env with production values
```

### 2. Build images

```sh
docker compose build
```

### 3. Start all services

```sh
docker compose up -d
```

### 4. Run migrations

```sh
make migrate
```

### 5. Verify

```sh
docker compose ps       # all services should be "running"
docker compose logs -f  # tail logs
```

The application is now accessible at `http://your-server-ip`.

### Useful commands

```sh
# Stop everything
docker compose down

# Stop and remove volumes (destructive — deletes all data)
docker compose down -v

# Restart a single service
docker compose restart api

# View logs for one service
docker compose logs -f api

# Pull latest images after a code update
docker compose build --no-cache && docker compose up -d
```

### Compose service map

| Service | Port (internal) | Description |
|---|---|---|
| `nginx` | 80 | Reverse proxy — public entry point |
| `api` | 8080 | Go HTTP API server |
| `worker` | — | Background job processor |
| `frontend` | 3000 | SvelteKit Node.js server |
| `postgres` | 5432 | PostgreSQL database |
| `redis` | 6379 | Redis cache / pub-sub |

---

## Method 2 — Manual Bare-Metal

Use this if you want to run the app directly on a server without Docker.

### Requirements on the server

- Go 1.22+
- Node.js 20+
- PostgreSQL 16+ running and accessible
- Redis 7+ running and accessible

### 1. Clone and configure

```sh
git clone https://github.com/404nfid/go-svelte-starter-kit.git
cd go-svelte-starter-kit
cp backend/.env.example backend/.env
# Edit backend/.env
```

### 2. Build the backend

```sh
cd backend
go build -ldflags="-s -w" -o bin/api ./cmd/api
go build -ldflags="-s -w" -o bin/worker ./cmd/worker
```

Binaries are output to `backend/bin/`.

### 3. Build the frontend

```sh
cd frontend
npm ci
npm run build
```

The production build is output to `frontend/build/`.

### 4. Run migrations

```sh
make migrate
```

### 5. Start the services

**API server:**

```sh
cd backend
export $(cat .env | xargs)   # load env vars
./bin/api
```

**Background worker** (separate terminal or process):

```sh
cd backend
export $(cat .env | xargs)
./bin/worker
```

**Frontend:**

```sh
cd frontend
NODE_ENV=production node build
```

The API listens on port `8080` (or `APP_PORT`), and the frontend on port `3000`.

---

## Method 3 — Manual with systemd (Linux)

For running the backend binaries as persistent system services on a Linux server (Ubuntu/Debian/RHEL).

### Assumptions

- Application cloned to `/opt/go-svelte-starter-kit`
- A dedicated `appuser` system account
- Binaries already built (see Method 2 steps 1–3)

### Create system user

```sh
sudo useradd --system --no-create-home --shell /bin/false appuser
sudo chown -R appuser:appuser /opt/go-svelte-starter-kit
```

### API service

Create `/etc/systemd/system/starterkit-api.service`:

```ini
[Unit]
Description=StarterKit API Server
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=appuser
WorkingDirectory=/opt/go-svelte-starter-kit/backend
EnvironmentFile=/opt/go-svelte-starter-kit/backend/.env
ExecStart=/opt/go-svelte-starter-kit/backend/bin/api
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=starterkit-api

[Install]
WantedBy=multi-user.target
```

### Worker service

Create `/etc/systemd/system/starterkit-worker.service`:

```ini
[Unit]
Description=StarterKit Background Worker
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=appuser
WorkingDirectory=/opt/go-svelte-starter-kit/backend
EnvironmentFile=/opt/go-svelte-starter-kit/backend/.env
ExecStart=/opt/go-svelte-starter-kit/backend/bin/worker
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=starterkit-worker

[Install]
WantedBy=multi-user.target
```

### Frontend service

Create `/etc/systemd/system/starterkit-frontend.service`:

```ini
[Unit]
Description=StarterKit Frontend
After=network.target

[Service]
Type=simple
User=appuser
WorkingDirectory=/opt/go-svelte-starter-kit/frontend
Environment=NODE_ENV=production
Environment=PORT=3000
ExecStart=/usr/bin/node build
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=starterkit-frontend

[Install]
WantedBy=multi-user.target
```

### Enable and start

```sh
sudo systemctl daemon-reload
sudo systemctl enable starterkit-api starterkit-worker starterkit-frontend
sudo systemctl start starterkit-api starterkit-worker starterkit-frontend

# Check status
sudo systemctl status starterkit-api
sudo journalctl -u starterkit-api -f
```

---

## Method 4 — Reverse Proxy with Nginx (no Docker)

If running the services manually (Methods 2–3), use nginx to expose a single port to the internet.

### Install nginx

```sh
# Ubuntu / Debian
sudo apt install nginx

# RHEL / AlmaLinux / Rocky
sudo dnf install nginx
```

### Configuration

Create `/etc/nginx/sites-available/starterkit`:

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    # Uploaded files served from Go backend
    location /uploads {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }

    # API and WebSocket
    location /api {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   Upgrade           $http_upgrade;
        proxy_set_header   Connection        "upgrade";
        proxy_read_timeout 3600s;
    }

    # Frontend
    location / {
        proxy_pass         http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }
}
```

```sh
sudo ln -s /etc/nginx/sites-available/starterkit /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## Method 5 — Fly.io

Fly.io can run Docker images directly. Deploy the backend API and frontend as separate Fly apps.

### Prerequisites

```sh
# Install flyctl
curl -L https://fly.io/install.sh | sh
fly auth login
```

### Deploy the API

```sh
cd backend
fly launch --name starterkit-api --dockerfile Dockerfile --build-target api
fly secrets set \
  DATABASE_URL="postgres://..." \
  REDIS_URL="redis://..." \
  JWT_ACCESS_SECRET="..." \
  JWT_REFRESH_SECRET="..." \
  APP_ENV=production
fly deploy
```

### Deploy the frontend

```sh
cd frontend
fly launch --name starterkit-frontend --dockerfile Dockerfile
fly secrets set NODE_ENV=production
fly deploy
```

### Managed Postgres on Fly

```sh
fly postgres create --name starterkit-db
fly postgres attach --app starterkit-api starterkit-db
```

### Managed Redis on Fly (Upstash)

```sh
fly ext upstash redis create --name starterkit-redis
# Copy the REDIS_URL from the output into your API secrets
```

### Run migrations on Fly

```sh
fly ssh console --app starterkit-api
# Inside the container:
migrate -path /app/migrations -database "$DATABASE_URL" up
```

---

## Method 6 — Railway

Railway supports Docker-based deploys with automatic HTTPS.

### Steps

1. Go to [railway.app](https://railway.app) and create a new project.
2. Click **Deploy from GitHub repo** and select this repository.
3. Railway auto-detects the `docker-compose.yml` and creates services.
4. Add a **PostgreSQL** plugin and a **Redis** plugin from the Railway dashboard.
5. In each service's **Variables** tab, add the required environment variables.
6. Set `DATABASE_URL` and `REDIS_URL` to the Railway-provided connection strings.
7. Run migrations via the Railway shell:
   ```sh
   migrate -path /app/migrations -database "$DATABASE_URL" up
   ```

Railway assigns a public URL to each service automatically.

---

## Method 7 — Render

### Backend (Web Service)

1. Create a new **Web Service** in the Render dashboard.
2. Connect your GitHub repository.
3. Set **Dockerfile path** to `backend/Dockerfile` and **Docker build target** to `api`.
4. Add environment variables from `backend/.env.example`.
5. Add a **PostgreSQL** database and a **Redis** instance from the Render dashboard.
6. Copy the connection strings into the Web Service environment variables.

### Worker (Background Worker)

1. Create a new **Background Worker** service.
2. Same repository, same Dockerfile, but set **build target** to `worker`.
3. Share the same environment variables as the API service.

### Frontend (Web Service)

1. Create another **Web Service**.
2. Set **Dockerfile path** to `frontend/Dockerfile`.
3. Set `NODE_ENV=production`.

### Migrations on Render

Use a **one-off job** or the Render shell on the API service:

```sh
migrate -path /app/migrations -database "$DATABASE_URL" up
```

---

## SSL / TLS (HTTPS)

### Docker Compose — with Certbot

Update `nginx.conf` to include your domain and obtain a certificate:

```sh
# Stop nginx temporarily
docker compose stop nginx

# Obtain certificate (standalone mode)
sudo certbot certonly --standalone -d yourdomain.com

# Update nginx.conf to reference the certificate paths
# Then restart
docker compose start nginx
```

Or use the `nginx:alpine` + `certbot/certbot` pattern with a volume mount:

```yaml
# In docker-compose.yml, add to nginx volumes:
  - /etc/letsencrypt:/etc/letsencrypt:ro
```

And update `nginx.conf`:

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # ... rest of your location blocks
}
```

### Manual Nginx — with Certbot

```sh
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
sudo systemctl reload nginx
```

Certbot automatically edits your nginx config and sets up auto-renewal.

### Auto-renewal

```sh
# Test renewal
sudo certbot renew --dry-run

# Certbot installs a systemd timer by default — verify it
sudo systemctl status certbot.timer
```

---

## Health Checks & Monitoring

The API exposes a health endpoint:

```
GET /api/health
```

Returns `200 OK` with service status. Use this for:
- Docker `HEALTHCHECK` directives
- Load balancer health probes
- Uptime monitoring (UptimeRobot, Betterstack, etc.)

### Docker health check (add to `docker-compose.yml`)

```yaml
api:
  healthcheck:
    test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/health"]
    interval: 30s
    timeout: 5s
    retries: 3
    start_period: 10s
```

### Viewing logs

```sh
# Docker Compose
docker compose logs -f api
docker compose logs -f worker
docker compose logs --since 1h api

# systemd
journalctl -u starterkit-api -f
journalctl -u starterkit-api --since "1 hour ago"
```

---

## Upgrading

### Docker Compose

```sh
git pull
docker compose build --no-cache
docker compose up -d
make migrate
```

### Bare-metal / systemd

```sh
git pull

# Rebuild
cd backend && go build -ldflags="-s -w" -o bin/api ./cmd/api && go build -ldflags="-s -w" -o bin/worker ./cmd/worker
cd ../frontend && npm ci && npm run build

# Run migrations
make migrate

# Restart services
sudo systemctl restart starterkit-api starterkit-worker starterkit-frontend
```

### Cloud platforms (Fly.io / Railway / Render)

Push to your connected branch — the platform triggers an automatic rebuild and redeploy. Run migrations manually via the platform shell after each deploy that includes schema changes.

---

## Checklist — Production Readiness

Before going live, verify the following:

- [ ] `APP_ENV=production` is set
- [ ] `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` are random strings of at least 32 characters
- [ ] `DATABASE_URL` points to a production database with `sslmode=require`
- [ ] All migrations have been applied (`make migrate`)
- [ ] HTTPS is configured and HTTP redirects to HTTPS
- [ ] `APP_URL` and `FRONTEND_URL` are set to the public HTTPS domain
- [ ] `GOOGLE_REDIRECT_URL` (if using OAuth) matches the public domain
- [ ] Rate limiting values (`RATE_LIMIT_AUTH`, `RATE_LIMIT_API`) are appropriate for expected load
- [ ] Logs are being collected and retained
- [ ] A database backup strategy is in place
- [ ] The `/api/health` endpoint is being monitored
