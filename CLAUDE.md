# CLAUDE.md

## General Rules & Code Style
- **Imports:** ALWAYS use absolute imports (`@/` in the Angular apps, `~/` in the remaining Nuxt apps) instead of relative paths (`../`, `./`).
- **Architecture:**
  - **Go services (`services/`):** Follow standard Gin conventions, not Clean Architecture — `cmd/<service>/main.go`, and under `internal/`: `config`, `routes` (router + middleware wiring), `handlers`, `middleware`, plus domain packages as needed. Shared code lives in `services/pkg`.
  - **Frontend:** Feature-Sliced Design (FSD) (`app/`, `pages/`, `widgets/`, `features/`, `entities/`, `shared/`). Respect strict cross-layer import rules.
- **Formatting & Linting:** Use **Biome** for formatting and linting (`pnpm exec biome check --write`); Angular templates are linted with `angular-eslint` (`ng lint`).
- **Validation:** go-playground/validator in the Go services (via `services/pkg/validation`), **Zod** in the Node bot until it moves to Go, **Signal Forms** (+ Zod schemas via Standard Schema) in the Angular apps, **VeeValidate** in the remaining Nuxt apps.
- **Logging:** `log/slog` in Go, **Pino** in the Node bot until it moves to Go. Avoid `console.log`.
- **Environment:** Manage environment variables via `dotenv` (`.env`).
- **TypeScript:** Strict mode enabled. Always specify return types for public functions/methods.

## Migrations in Progress
Plan, phases and requirements: `docs/angular-migration-plan.md`. Work on either track starts only on an explicit request.

**Track A — frontend, Nuxt 3 → Angular 22:**
- **Apps:** `apps/site` (public site + Telegram Mini App, Angular SSR) replaces `apps/tourism`; `apps/panel` (admin, client-only) replaces `apps/admin`. nginx routes already-migrated paths to the Angular app and the rest to Nuxt until each wave is switched.
- **New features:** in migrated areas only in Angular; in areas not yet migrated only small fixes in Nuxt. Don't add new Nuxt modules or dependencies.
- **URLs are a contract:** keep every public path and the locale scheme (Russian unprefixed, `/ky`, `/en`, `/de`) exactly as they are.

**Track B — Telegram bot, NestJS + Telegraf (`apps/bot`) → Go + [Telego](https://github.com/mymmrac/telego) (`services/bot`):**
- Its own Docker service (`services/bot/Dockerfile`, image `ermekten-bot`); same Go conventions as the other services, but Telego (`telego`, `telegohandler`, `telegoutil`) instead of Gin — the bot has no HTTP API. Layout: `cmd/bot/main.go`, `internal/{config,handlers,keyboards,session}`.
- Long polling, as today; users data over the existing `users` gRPC API; same env (`BOT_TOKEN`, `USERS_GRPC_ADDR`, `MINI_APP_URL`).
- Behavior parity: same commands, deep links (`verify_`, `link_`), callback data, message texts and keyboards. No new bot features in `apps/bot`.
- `trails`/`payments` keep sending notifications through `services/pkg/telegram`.

## Angular Apps
- **Stack:** Angular 22 standalone components, zoneless, `OnPush`, signals (`signal`/`computed`/`input`/`output`/`model`), `inject()`, built-in control flow (`@if`/`@for`/`@defer`), `HttpClient` (fetch) with `httpResource`/`resource`, Signal Forms, Transloco (the existing `ru/ky/en/de.json` files), Tailwind CSS 4 + Iconify, Angular CDK/Aria under the in-house `shared/ui` kit, Vitest.
- **Layout:** `src/app` (config, `app.routes.ts`, `app.routes.server.ts`), `src/pages`, `src/widgets`, `src/features`, `src/entities`, `src/shared` (`api`, `ui`, `lib`, `seo`).
- **Rendering (`apps/site`):** set every route's mode in `app.routes.server.ts` — `RenderMode.Server` for public catalog and detail pages (anonymous render, cacheable HTML), `RenderMode.Prerender` for static pages, `RenderMode.Client` + `noindex` for account, cabinet, payment and edit pages. `sitemap.xml` and `robots.txt` are served by the SSR server.
- **SSR safety:** no `window`/`document`/`localStorage`/Telegram WebApp access outside `afterNextRender`, `@defer` or an `isPlatformBrowser` check. Server-rendered data reaches the client through the HTTP transfer cache — never fetch it twice.
- **SEO on every public page** (through `shared/seo`): title and description (admin `/page-seo` overrides win), absolute canonical, `hreflang` for all locales + `x-default`, Open Graph/Twitter, JSON-LD for the page type plus `BreadcrumbList`, correct HTTP status (404 for unknown slugs), `NgOptimizedImage` with dimensions and an LCP `priority` image, heavy widgets (maps, reviews, booking, charts, editor) behind `@defer` with incremental hydration.

## Go Services
- **Stack:** Go (version in each `go.mod`), Gin, `log/slog` JSON logging (`services/pkg/logger`), `go-playground/validator` via Gin binding, config from env via `services/pkg/config` (`.env` supported).
- **API contract:** The frontend apps (Nuxt and Angular) depend on the existing paths, cookies (`user_token`, `admin_token`), JSON shapes and NestJS-style error bodies (`services/pkg/httperr`, zod-style validation errors from `services/pkg/validation`). Keep them stable.
- **Lint/Test:** `golangci-lint run ./...` per module (config in `services/.golangci.yml`), `go test ./...`. Imports: stdlib, third-party, then `github.com/elvn5/sai/...`.
- **Databases:** Each service owns its own PostgreSQL database (created on first start), with goose SQL migrations embedded from `internal/migrations` and pgx for queries. No cross-database foreign keys — reference other services' records by id.
- **Inter-service calls:** gRPC; contracts in `services/proto` (buf). Run `pnpm proto:generate` after editing a `.proto` and commit the generated code.
- **Modules:** One Go module per service; local `pkg` is wired with a `replace` directive so each service builds alone (`GOWORK=off`) in Docker.

## Tech Stack
- **Backend:** Go + Gin (`services/`), gRPC (buf), PostgreSQL (pgx, goose), Docker
- **Bot:** Node.js v24, NestJS + Telegraf (`apps/bot`), gRPC client to the users service — moving to Go + Telego (`services/bot`), see Migrations in Progress
- **Frontend (target):** Angular 22 + `@angular/ssr` (`apps/site`, `apps/panel`) — see Angular Apps above
- **Frontend (legacy, until each app is migrated):** Nuxt 3, Pinia (+ persistedstate), @nuxt/ui, Tailwind CSS, Nuxt Icon, @nuxtjs/seo, @nuxtjs/i18n, @vueuse/nuxt, @tanstack/vue-query (`apps/tourism`, `apps/admin`)
- **Linter/Formatter:** Biome (+ angular-eslint for Angular templates)

## Commands
- **Dev:** `pnpm dev:<app>` (bot, admin, tourism; site and panel once scaffolded in migration phase 0) / `pnpm dev:<service>` (gateway, media, content, listings, users, trails, payments)
- **Build:** `pnpm build`
- **Lint/Format:** `pnpm exec biome check --write .`
- **Type Check:** `pnpm typecheck`
- **Docker:** `docker compose up -d --build`
- **Go:** `pnpm dev:gateway`, `pnpm test:go`
