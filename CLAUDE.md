# CLAUDE.md

## General Rules & Code Style
- **Imports:** ALWAYS use absolute imports (`@/` or `~/`) instead of relative paths (`../`, `./`).
- **Architecture:**
  - **Go services (`services/`):** Follow standard Gin conventions, not Clean Architecture — `cmd/<service>/main.go`, and under `internal/`: `config`, `routes` (router + middleware wiring), `handlers`, `middleware`, plus domain packages as needed. Shared code lives in `services/pkg`.
  - **Backend (NestJS, legacy):** Clean Architecture (Entities -> Use Cases -> Controllers / Interface Adapters -> Frameworks & Drivers). Keep business logic strictly independent of frameworks/DBs.
  - **Frontend:** Feature-Sliced Design (FSD) (`app/`, `pages/`, `widgets/`, `features/`, `entities/`, `shared/`). Respect strict cross-layer import rules.
- **Formatting & Linting:** Use **Biome** for formatting and linting (`npx @biomejs/biome check --write`).
- **Validation:** Use **Zod** (backend) and **VeeValidate** (frontend) for input/data validation.
- **Logging:** Use **Pino** for all logging. Avoid `console.log`.
- **Environment:** Manage environment variables via `dotenv` (`.env`).
- **TypeScript:** Strict mode enabled. Always specify return types for public functions/methods.

## Go Services
- **Stack:** Go (version in each `go.mod`), Gin, `log/slog` JSON logging (`services/pkg/logger`), `go-playground/validator` via Gin binding, config from env via `services/pkg/config` (`.env` supported).
- **API compatibility:** The Nuxt apps must not notice a migration. Keep paths, cookies (`user_token`, `admin_token`), JSON shapes and NestJS-style errors (`services/pkg/httperr`) identical.
- **Lint/Test:** `golangci-lint run ./...` per module (config in `services/.golangci.yml`), `go test ./...`. Imports: stdlib, third-party, then `github.com/elvn5/sai/...`.
- **Databases:** Each service owns its own PostgreSQL database (created on first start), with goose SQL migrations embedded from `internal/migrations` and pgx for queries. No cross-database foreign keys — reference other services' records by id.
- **Inter-service calls:** gRPC; contracts in `services/proto` (buf). Run `pnpm proto:generate` after editing a `.proto` and commit the generated code.
- **Modules:** One Go module per service; local `pkg` is wired with a `replace` directive so each service builds alone (`GOWORK=off`) in Docker.

## Tech Stack
- **Backend:** Go + Gin (`services/`, new code), Node.js v24, Next.js / Express, TypeORM, PostgreSQL, Axios, Socket.io, CORS, Docker
- **Frontend:** Nuxt 3, Pinia (+ persistedstate), @nuxt/ui, Tailwind CSS, Nuxt Icon, @nuxtjs/seo, @nuxtjs/i18n, @vueuse/nuxt, @tanstack/vue-query
- **Linter/Formatter:** Biome

## Commands
- **Dev:** `npm run dev` / `pnpm dev`
- **Build:** `npm run build` / `pnpm build`
- **Lint/Format:** `npx @biomejs/biome check --write .`
- **Type Check:** `npx tsc --noEmit`
- **Docker:** `docker compose up -d --build`
- **Go:** `pnpm dev:gateway`, `pnpm test:go`
