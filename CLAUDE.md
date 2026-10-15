# CLAUDE.md

## General Rules & Code Style
- **Imports:** ALWAYS use absolute imports (`@/` in the Angular apps, `~/` in the remaining Nuxt apps) instead of relative paths (`../`, `./`).
- **Architecture:**
  - **Go services (`services/`):** Follow standard Gin conventions, not Clean Architecture — `cmd/<service>/main.go`, and under `internal/`: `config`, `routes` (router + middleware wiring), `handlers`, `middleware`, plus domain packages as needed. Shared code lives in `services/pkg`.
  - **Angular apps:** folders by role and area, not Feature-Sliced Design — see Angular Apps → Layout. The legacy Nuxt apps keep their existing `components/` folders until they are removed.
- **Formatting & Linting:** Use **Biome** for formatting and linting (`pnpm exec biome check --write`); Angular templates are linted with `angular-eslint` (`ng lint`).
- **Validation:** go-playground/validator in the Go services (via `services/pkg/validation`), **Signal Forms** (+ Zod schemas via Standard Schema) in the Angular apps, **VeeValidate** in the remaining Nuxt apps.
- **Logging:** `log/slog` in Go. Avoid `console.log`.
- **Environment:** Manage environment variables via `dotenv` (`.env`).
- **TypeScript:** Strict mode enabled. Always specify return types for public functions/methods.

## Migrations in Progress
Plan, phases and requirements: `docs/angular-migration-plan.md`. Work on either track starts only on an explicit request.

**Track A — frontend, Nuxt 3 → Angular 22:**
- **Apps:** `apps/site` (public site + Telegram Mini App, Angular SSR) replaces `apps/tourism`; `apps/panel` (admin, client-only) replaces `apps/admin`. Staging (test.ermekten.com, test.admin.ermekten.com) runs the Angular apps instead of Nuxt; production runs Nuxt, and nginx routes already-migrated paths to the Angular app as each wave is switched.
- **New features:** in migrated areas only in Angular; in areas not yet migrated only small fixes in Nuxt. Don't add new Nuxt modules or dependencies.
- **URLs are a contract:** keep every public path and the locale scheme (Russian unprefixed, `/ky`, `/en`, `/de`) exactly as they are.

**Track B — Telegram bot, NestJS + Telegraf (`apps/bot`) → Go + [Telego](https://github.com/mymmrac/telego) (`services/bot`):** the Go bot is in place and deployed instead of `apps/bot`; `apps/bot` stays only until the switch is verified in production (step B5 removes it).
- Its own Docker service (`services/bot/Dockerfile`, image `ermekten-bot`); same Go conventions as the other services, but Telego (`telego`, `telegohandler`, `telegoutil`) instead of Gin — the bot has no HTTP API. Layout: `cmd/bot/main.go`, `internal/{config,handlers,keyboards,session,users}`.
- Long polling, as today; users data over the existing `users` gRPC API; same env (`BOT_TOKEN`, `USERS_GRPC_ADDR`, `MINI_APP_URL`).
- Behavior parity: same commands, deep links (`verify_`, `link_`), callback data, message texts and keyboards. Bot changes go to `services/bot` only.
- `trails`/`payments` keep sending notifications through `services/pkg/telegram`.

## Angular Apps
- **Stack:** Angular 22 standalone components, zoneless, `OnPush`, signals (`signal`/`computed`/`input`/`output`/`model`), `inject()`, built-in control flow (`@if`/`@for`/`@defer`), `HttpClient` (fetch) with `httpResource`/`resource`, Signal Forms with `zod/mini` schemas (`validateStandardSchema`; the full `zod` build adds ~70 kB gzipped to the browser), Transloco (the Nuxt site's `apps/tourism/i18n/locales/*.json` via the `@locales/*` path until it's gone; `{param}` interpolation), Tailwind CSS 4 + Iconify (`icon-[heroicons--map]`), design tokens from `packages/tokens`, Angular CDK/Aria under the in-house `ui/` kit, Vitest. Node from `.nvmrc` (`nvm use`; Angular CLI 22 needs ≥ 24.15).
- **Layout:** `src/app` (bootstrap, config, `app.routes.ts`, `app.routes.server.ts`, root component); `src/core` (app-wide services and helpers by topic: `api`, `config`, `auth`, `i18n`, `seo`, `lib`…); `src/layout` (the page frame: header, footer, sidebar and their controls); `src/ui` (reusable components and directives); `src/pages/<area>` (route components plus anything only that area uses). Dependencies point inward: `pages` → `layout`/`ui` → `core`; `core` and `ui` never import `pages`. Move something to `ui`/`core` once a second area needs it.
- **Runtime config, not build-time:** one image serves every environment. `apps/site` reads `API_BASE_URL`, `SITE_URL`, `TELEGRAM_BOT_USERNAME` (and the server-only `API_INTERNAL_URL`, the gateway on the Docker network for server-side requests) in `src/server.ts` and hands them to the app via the request context + TransferState (`APP_CONFIG`); `apps/panel` loads `/config.json`, written from `API_BASE_URL` when its nginx container starts.
- **Data and forms:** read with `httpResource` (reactive URL, `reload()` after a change), write with `HttpClient`; custom form controls implement `FormValueControl` (`value = model()`), and limits like max length go into the form schema (`maxLength`, `min`), not template attributes. Icons whose names come from the API (`i-mdi-trophy`) render through `app-icon` (Iconify web component); static ones use the CSS classes.
- **apps/panel UI kit:** `ui-*` classes in `src/styles.css` (buttons, fields, tables, badges, dialogs), dialogs via Angular CDK `Dialog`, `src/ui` for the image upload field, the Tiptap editor and icons.
- **Rendering (`apps/site`):** set every route's mode in `app.routes.server.ts` — `RenderMode.Server` for public catalog and detail pages (anonymous render, cacheable HTML), `RenderMode.Prerender` for static pages, `RenderMode.Client` + `noindex` for account, cabinet, payment and edit pages. `sitemap.xml` (an index of `/sitemaps/<type>.xml`, `src/server/sitemap.ts` — add new content types there) and `robots.txt` are served by the SSR server.
- **Links (`apps/site`):** internal links use `appSiteLink="/tours"` (`ui/site-link.ts`): it adds the language prefix and routes inside the app only for sections listed in `core/config/sections.ts` (`MIGRATED_SECTIONS`); other paths load from the server, because production nginx still sends them to Nuxt. A wave adds its sections there, in `apps/tourism/plugins/migrated-sections.client.ts` and in the nginx config (`docs/strangler-nginx.md`); `scripts/compare-crawl.mjs` must pass before the switch. Session state comes from `core/auth/auth.store.ts` (`AuthStore.ensureSession()`, browser only); `requireAuth`/`requirePartner` guard Client-rendered routes.
- **Page states (`apps/site`):** an unknown slug renders `NotFoundPage` (status 404); a failed API call renders `ui/load-error` (status 503 via `core/seo/response-status.ts`, which nginx doesn't cache). Browser-only routes (RenderMode.Client) get their runtime config from `/app-config.json`, served by the SSR server.
- **SSR safety:** no `window`/`document`/`localStorage`/Telegram WebApp access outside `afterNextRender`, `@defer` or an `isPlatformBrowser` check. Server-rendered data reaches the client through the HTTP transfer cache — never fetch it twice. That data is the anonymous view (the server sends no cookies): per-user fields (`isJoined`, own bookings) must be re-fetched in the browser after hydration (with `transferCache: false` when it's the same URL, or the cache hands back the server's anonymous answer).
- **SEO on every public page** (through `core/seo`): title and description (admin `/page-seo` overrides win), absolute canonical, `hreflang` for all locales + `x-default`, Open Graph/Twitter, JSON-LD for the page type plus `BreadcrumbList`, correct HTTP status (404 for unknown slugs), `NgOptimizedImage` with dimensions and an LCP `priority` image, heavy widgets (maps, reviews, booking, charts, editor) behind `@defer` with incremental hydration — `hydrate on viewport`/`idle` for anything that depends on the signed-in visitor (never `on interaction`: until then the server's anonymous HTML stays, and the click is replayed on it). Deferred components need a box of their own (`host: { class: 'block' }`) for the viewport trigger.

## Go Services
- **Stack:** Go (version in each `go.mod`), Gin, `log/slog` JSON logging (`services/pkg/logger`), `go-playground/validator` via Gin binding, config from env via `services/pkg/config` (`.env` supported).
- **API contract:** The frontend apps (Nuxt and Angular) depend on the existing paths, cookies (`user_token`, `admin_token`), JSON shapes and NestJS-style error bodies (`services/pkg/httperr`, zod-style validation errors from `services/pkg/validation`). Keep them stable.
- **Lint/Test:** `golangci-lint run ./...` per module (config in `services/.golangci.yml`), `go test ./...`. Imports: stdlib, third-party, then `github.com/elvn5/sai/...`.
- **Databases:** Each service owns its own PostgreSQL database (created on first start), with goose SQL migrations embedded from `internal/migrations` and pgx for queries. No cross-database foreign keys — reference other services' records by id.
- **Inter-service calls:** gRPC; contracts in `services/proto` (buf). Run `pnpm proto:generate` after editing a `.proto` and commit the generated code.
- **Modules:** One Go module per service; local `pkg` is wired with a `replace` directive so each service builds alone (`GOWORK=off`) in Docker.

## Tech Stack
- **Backend:** Go + Gin (`services/`), gRPC (buf), PostgreSQL (pgx, goose), Docker
- **Bot:** Go + Telego (`services/bot`), long polling, gRPC client to the users service (the NestJS `apps/bot` is legacy, see Migrations in Progress)
- **Frontend (target):** Angular 22 + `@angular/ssr` (`apps/site`, `apps/panel`) — see Angular Apps above
- **Frontend (legacy, until each app is migrated):** Nuxt 3, Pinia (+ persistedstate), @nuxt/ui, Tailwind CSS, Nuxt Icon, @nuxtjs/seo, @nuxtjs/i18n, @vueuse/nuxt, @tanstack/vue-query (`apps/tourism`, `apps/admin`)
- **Linter/Formatter:** Biome (+ angular-eslint for Angular templates)

## Commands
- **Dev:** `pnpm dev:<app>` (admin, tourism, site on :3012, panel on :3013) / `pnpm dev:<service>` (gateway, media, content, listings, users, trails, payments, bot)
- **Build:** `pnpm build`
- **Lint/Format:** `pnpm exec biome check --write .`
- **Type Check:** `pnpm typecheck`
- **Angular apps:** `pnpm --filter @app/site --filter @app/panel lint` (templates), `… test` (Vitest); Lighthouse budgets in `apps/site/lighthouserc.json` (CI job `lighthouse`)
- **Docker:** `docker compose up -d --build` (`--profile angular` adds the Angular apps locally on :3012/:3013; deploys pick `nuxt` in production, `angular` on staging)
- **Go:** `pnpm dev:gateway`, `pnpm test:go`
