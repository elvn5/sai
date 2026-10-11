# Ermekten

Ermekten: a travel site for Kyrgyzstan with an admin panel and a Telegram bot. The backend is
a set of Go (Gin) services behind an API gateway, each with its own PostgreSQL database; the
frontends are Nuxt 3 and the bot is NestJS + Telegraf.

## Stack

| App | Framework | Purpose |
| --- | --- | --- |
| [`services/gateway`](services/gateway) | Go + Gin | Public API entry point; routes requests to backend services |
| [`services/media`](services/media) | Go + Gin | Uploads: R2 presigned URLs, WebP image conversion, `/uploads` files |
| [`services/content`](services/content) | Go + Gin | Blog, written guides, gallery, page SEO; own `content` database |
| [`services/listings`](services/listings) | Go + Gin | Hotels, local guides, tours, reviews, owner posts; own `listings` database |
| [`services/trails`](services/trails) | Go + Gin | Route catalog with GPX tracks, trail condition reports, group outings; own `trails` database |
| [`services/users`](services/users) | Go + Gin | Accounts: sign-in (password, Telegram, admin), registration, partners, support; gRPC for the other services and the bot; own `users` database |
| [`apps/bot`](apps/bot) | NestJS + Telegraf | Telegram bot (no database; accounts via the users service's gRPC API) |
| [`apps/tourism`](apps/tourism) | Nuxt 3 + @nuxt/ui | Public site and Telegram Mini App |
| [`apps/admin`](apps/admin) | Nuxt 3 + @nuxt/ui | Admin panel (Pinia, VeeValidate, Zod) |

Services talk to each other over gRPC (contracts in [`services/proto`](services/proto)).
TypeScript is formatted/linted with [Biome](https://biomejs.dev), Go with golangci-lint.

## Requirements

- Node.js >= 24
- pnpm 9 (`corepack enable`)
- Go >= 1.26 (for `services/`)
- Docker + Docker Compose (for PostgreSQL and/or full-stack runs)

Each Go service creates its database on the Postgres server on first start and applies its
own migrations (goose, embedded in the binary) — there is no separate migration step.

## Getting started

```bash
corepack enable
pnpm install
cp .env.example .env
```

Fill in `.env` — at minimum set `POSTGRES_PASSWORD`, `JWT_SECRET`, `ADMIN_PASSWORD`, and
`BOT_TOKEN` (from [@BotFather](https://t.me/BotFather)). See [.env.example](.env.example)
for the full list.

### Run with Docker (recommended)

```bash
docker compose up -d --build
```

This starts PostgreSQL, the Go services with the gateway on the public API port (`:3001`), the
bot, the site (`:3002`) and the admin panel (`:3000`).

### Run locally

Start PostgreSQL (e.g. `docker compose up -d postgres`), then in separate terminals:

```bash
pnpm dev:users    # http://localhost:8084 + gRPC :9083
pnpm dev:media    # http://localhost:8081 + gRPC :9081
pnpm dev:content  # http://localhost:8082
pnpm dev:listings # http://localhost:8083
pnpm dev:trails   # http://localhost:8085
pnpm dev:gateway  # http://localhost:8080 — the API the Nuxt apps call
pnpm dev:bot      # needs USERS_GRPC_ADDR=localhost:9083
pnpm dev:tourism  # http://localhost:3011
pnpm dev:admin    # http://localhost:3000
```

Point the Nuxt apps at the gateway with `NUXT_PUBLIC_API_BASE_URL=http://localhost:8080`.
The services read their settings from `.env` (see [.env.example](.env.example)).

## Scripts

| Command | Description |
| --- | --- |
| `pnpm dev:bot` / `dev:admin` / `dev:tourism` | Run an app in watch mode |
| `pnpm build` | Build all workspace packages |
| `pnpm lint` / `pnpm lint:fix` | Biome check / check --write |
| `pnpm typecheck` | Type-check all workspace packages |
| `pnpm dev:<service>` (gateway, media, content, listings, users, trails) | Run a Go service |
| `pnpm proto:generate` | Regenerate gRPC code from `services/proto` |
| `pnpm test:go` | Test all Go modules |

## Project structure

```
services/   Go workspace (go.work)
  gateway/  Gin API gateway
  media/    Uploads service (R2, WebP, /uploads)
  content/  Blog, guides, gallery, page SEO (own database)
  listings/ Hotels, local guides, tours, reviews, posts (own database)
  trails/   Routes, condition reports, group outings (own database)
  users/    Accounts, sign-in, partners, support (own database; HTTP + gRPC)
  proto/    gRPC contracts between services (buf; `pnpm proto:generate`)
  pkg/      Shared Go code: config, logging, errors, validation, CORS, JWT, databases
apps/
  bot/      NestJS Telegram bot (Telegraf; accounts via users gRPC)
  tourism/  Nuxt 3 public site
  admin/    Nuxt 3 admin panel
```

The Go services follow standard Gin project layout; the frontends follow Feature-Sliced Design. See [CLAUDE.md](CLAUDE.md) for full
conventions.

## Using this as a template

1. Click **Use this template** on GitHub to create a new repository.
2. Rename `parcel-bot-monorepo` in [package.json](package.json) and update this README.
3. Copy `.env.example` to `.env` and fill in real secrets — never commit `.env`.
4. Adjust `services/*` and `apps/*` to fit your domain; each service's schema lives in its
   `internal/migrations`.

## License

[MIT](LICENSE)
