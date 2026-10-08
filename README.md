# Parcel Bot Monorepo

A pnpm/TypeScript monorepo template for a Telegram bot + admin panel + REST API, built on
NestJS, Nuxt 3, and TypeORM/PostgreSQL. Use this repository as a **GitHub template**
("Use this template" button) to bootstrap similar bot + admin panel projects.

## Stack

| App | Framework | Purpose |
| --- | --- | --- |
| [`services/gateway`](services/gateway) | Go + Gin | Public API entry point; routes requests to backend services |
| [`services/media`](services/media) | Go + Gin | Uploads: R2 presigned URLs, WebP image conversion, `/uploads` files |
| [`apps/api`](apps/api) | NestJS | Legacy REST API, being migrated to Go services behind the gateway |
| [`apps/bot`](apps/bot) | NestJS + Telegraf | Telegram bot |
| [`apps/admin`](apps/admin) | Nuxt 3 + @nuxt/ui | Admin panel (Pinia, VeeValidate, Zod) |
| [`packages/database`](packages/database) | TypeORM | Shared entities, migrations, data source |

Formatting/linting via [Biome](https://biomejs.dev), logging via [Pino](https://getpino.io),
validation via [Zod](https://zod.dev) (backend) and [VeeValidate](https://vee-validate.logaretm.com) (frontend).

## Requirements

- Node.js >= 24
- pnpm 9 (`corepack enable`)
- Go >= 1.26 (for `services/`)
- Docker + Docker Compose (for PostgreSQL and/or full-stack runs)

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

This starts PostgreSQL, the gateway on the public API port (`:3001`) in front of the NestJS API
(internal only), the bot, and the admin panel (`:3000`).

### Run locally

Start PostgreSQL (e.g. `docker compose up -d postgres`), then in separate terminals:

```bash
pnpm dev:api     # http://localhost:3001
pnpm dev:gateway # http://localhost:8080, proxies to the API above
pnpm dev:media   # http://localhost:8081 (set MEDIA_URL for the gateway to use it)
pnpm dev:bot
pnpm dev:admin   # http://localhost:3000
```

Each `dev:*` script builds `@app/database` first so entity/migration changes are picked up.
The Nuxt apps can call the API directly (`:3001`) or through the gateway
(`NUXT_PUBLIC_API_BASE_URL=http://localhost:8080`).

## Scripts

| Command | Description |
| --- | --- |
| `pnpm dev:api` / `dev:bot` / `dev:admin` | Run an app in watch mode |
| `pnpm build` | Build all workspace packages |
| `pnpm lint` / `pnpm lint:fix` | Biome check / check --write |
| `pnpm typecheck` | Type-check all workspace packages |
| `pnpm migration:generate` | Generate a TypeORM migration (`packages/database`) |
| `pnpm migration:run` | Run pending migrations |
| `pnpm dev:gateway` / `dev:media` | Run a Go service |
| `pnpm test:go` | Test all Go modules |

## Project structure

```
services/   Go workspace (go.work)
  gateway/  Gin API gateway
  media/    Uploads service (R2, WebP, /uploads)
  pkg/      Shared Go code: config, logging, NestJS-compatible errors, CORS, JWT
apps/
  api/      NestJS REST API (legacy, behind the gateway)
  bot/      NestJS Telegram bot (Telegraf)
  admin/    Nuxt 3 admin panel
packages/
  database/ TypeORM entities, migrations, data source
```

The Go services follow standard Gin project layout; the NestJS API follows Clean Architecture;
the frontends follow Feature-Sliced Design. See [CLAUDE.md](CLAUDE.md) for full
conventions.

## Using this as a template

1. Click **Use this template** on GitHub to create a new repository.
2. Rename `parcel-bot-monorepo` in [package.json](package.json) and update this README.
3. Copy `.env.example` to `.env` and fill in real secrets — never commit `.env`.
4. Adjust `apps/*` to fit your domain; `packages/database` holds the shared schema.

## License

[MIT](LICENSE)
