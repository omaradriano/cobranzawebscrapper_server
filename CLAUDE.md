# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server
go run ./cmd/app/main.go

# Build
go build -o webscrapper_server ./cmd/app

# Run tests
go test ./...

# Format code
go fmt ./...
```

The server listens on the host/port defined in `.env` (default `127.0.0.1:3006`).

## Architecture

This is a Go HTTP API server for insurance policy (póliza) management used by insurance agents. It handles policy scraping/storage, billing (cobranzas), Stripe subscriptions, and agent authentication.

**Entry point:** `cmd/app/main.go` — loads env, connects to DB, starts the router.

**Custom router:** `configs/routing.go` implements a regex-based router (no external mux library). Routes are matched sequentially; middleware is applied per-route via `jwt_middleware.go`.

**Request flow:**
1. `configs/routing.go` matches URL pattern + HTTP method
2. Optionally wraps handler with `middlewares.JwtMiddleware` (injects agent data into context)
3. `internal/handlers/` handle the request logic
4. `internal/services/responseHandling.go` provides helpers for JSON responses, CORS headers, JWT generation, and bcrypt

**Packages:**
- `internal/handlers/` — HTTP handlers by domain: `auth.go`, `scrapping.go`, `cobranzas.go`, `stripe_payment.go`
- `internal/services/` — Shared utilities: response formatting, email (Resend API), logging
- `internal/structs.go` — All DTOs and domain models
- `db/dbConn.go` — PostgreSQL connection (raw SQL, no ORM)
- `env/envconf.go` — Loads `.env` via `godotenv`; all vars required or server fatals

**Key integrations:**
- **PostgreSQL** — primary store for agents (`agentes`) and policies (`polizas`)
- **Stripe** — subscription checkout and webhook (`stripe_payment.go`)
- **Resend** — transactional email (`internal/services/emailManager.go`)
- **Google OAuth** — agent login via Google token verification
- **JWT** — 24-hour session tokens; secret from `JWT_SECRET` env var

## Environment Variables

All variables are required (see `env/envconf.go`):

| Variable | Purpose |
|---|---|
| `DB_URL` | PostgreSQL connection string |
| `MODE` | `dev` or `prod` |
| `JWT_SECRET` | JWT signing secret |
| `SERVER_HOST`, `SERVER_PORT` | Bind address |
| `WEBAPP_URL` | Frontend origin for CORS |
| `TOKEN_RESEND` | Resend API key |
| `GOOGLE_API_AUTH_URL` | Google OAuth token verification URL |
| `STRIPE_SECRET`, `STRIPE_WEBHOOK_SECRET` | Stripe keys |
| `MAIL_DESTINATION_WEB`, `MAIL_DESTINATION_SERVER` | Email callback base URLs |

## Database

Schema is in `webscrapperdb.sql`. Raw SQL queries throughout — no ORM. The main tables are `agentes` (users) and `polizas` (insurance policies).

## Notes

- CORS allowed origins are hardcoded in `internal/services/responseHandling.go`: `localhost:5173`, `goagent.com.mx`, and two Chrome extension IDs.
- `go.mod` references both `stripe-go/v74` and `stripe-go/v76` — handlers use v76.
