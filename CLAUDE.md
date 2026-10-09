# CLAUDE.md

## General Rules & Code Style
- **Imports:** ALWAYS use absolute imports (`@/` or `~/`) instead of relative paths (`../`, `./`).
- **Architecture:**
  - **Go services (`services/`):** Follow standard Gin conventions, not Clean Architecture — `cmd/<service>/main.go`, and under `internal/`: `config`, `routes` (router + middleware wiring), `handlers`, `middleware`, plus domain packages as needed. Shared code lives in `services/pkg`.
  - **Frontend:** Feature-Sliced Design (FSD) (`app/`, `pages/`, `widgets/`, `features/`, `entities/`, `shared/`). Respect strict cross-layer import rules.
- **Formatting & Linting:** Use **Biome** for formatting and linting (`npx @biomejs/biome check --write`).
- **Validation:** go-playground/validator in the Go services (via `services/pkg/validation`), **Zod** in the bot, **VeeValidate** on the frontend.
- **Logging:** `log/slog` in Go, **Pino** in the bot. Avoid `console.log`.
- **Environment:** Manage environment variables via `dotenv` (`.env`).
- **TypeScript:** Strict mode enabled. Always specify return types for public functions/methods.

## Go Services
- **Stack:** Go (version in each `go.mod`), Gin, `log/slog` JSON logging (`services/pkg/logger`), `go-playground/validator` via Gin binding, config from env via `services/pkg/config` (`.env` supported).
- **API contract:** The Nuxt apps depend on the existing paths, cookies (`user_token`, `admin_token`), JSON shapes and NestJS-style error bodies (`services/pkg/httperr`, zod-style validation errors from `services/pkg/validation`). Keep them stable.
- **Lint/Test:** `golangci-lint run ./...` per module (config in `services/.golangci.yml`), `go test ./...`. Imports: stdlib, third-party, then `github.com/elvn5/sai/...`.
- **Databases:** Each service owns its own PostgreSQL database (created on first start), with goose SQL migrations embedded from `internal/migrations` and pgx for queries. No cross-database foreign keys — reference other services' records by id.
- **Inter-service calls:** gRPC; contracts in `services/proto` (buf). Run `pnpm proto:generate` after editing a `.proto` and commit the generated code.
- **Modules:** One Go module per service; local `pkg` is wired with a `replace` directive so each service builds alone (`GOWORK=off`) in Docker.

## Tech Stack
- **Backend:** Go + Gin (`services/`), gRPC (buf), PostgreSQL (pgx, goose), Docker
- **Bot:** Node.js v24, NestJS + Telegraf (`apps/bot`), gRPC client to the users service
- **Frontend:** Nuxt 3, Pinia (+ persistedstate), @nuxt/ui, Tailwind CSS, Nuxt Icon, @nuxtjs/seo, @nuxtjs/i18n, @vueuse/nuxt, @tanstack/vue-query
- **Linter/Formatter:** Biome

## Commands
- **Dev:** `pnpm dev:<app>` (bot, admin, tourism) / `pnpm dev:<service>` (gateway, media, content, listings, users)
- **Build:** `npm run build` / `pnpm build`
- **Lint/Format:** `npx @biomejs/biome check --write .`
- **Type Check:** `pnpm typecheck`
- **Docker:** `docker compose up -d --build`
- **Go:** `pnpm dev:gateway`, `pnpm test:go`
