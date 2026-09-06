# Parcel Bot Monorepo

A pnpm/TypeScript monorepo template for a Telegram bot + admin panel + REST API, built on
NestJS, Nuxt 3, and TypeORM/PostgreSQL. Use this repository as a **GitHub template**
("Use this template" button) to bootstrap similar bot + admin panel projects.

## Stack

| App | Framework | Purpose |
| --- | --- | --- |
| [`apps/api`](apps/api) | NestJS | REST API, JWT auth, TypeORM data access |
| [`apps/bot`](apps/bot) | NestJS + Telegraf | Telegram bot |
| [`apps/admin`](apps/admin) | Nuxt 3 + @nuxt/ui | Admin panel (Pinia, VeeValidate, Zod) |
| [`packages/database`](packages/database) | TypeORM | Shared entities, migrations, data source |

Formatting/linting via [Biome](https://biomejs.dev), logging via [Pino](https://getpino.io),
validation via [Zod](https://zod.dev) (backend) and [VeeValidate](https://vee-validate.logaretm.com) (frontend).

## Requirements

- Node.js >= 24
- pnpm 9 (`corepack enable`)
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

This starts PostgreSQL, the API (`:3001`), the bot, and the admin panel (`:3000`).

### Run locally

Start PostgreSQL (e.g. `docker compose up -d postgres`), then in separate terminals:

```bash
pnpm dev:api     # http://localhost:3001
pnpm dev:bot
pnpm dev:admin   # http://localhost:3000
```

Each `dev:*` script builds `@app/database` first so entity/migration changes are picked up.

## Scripts

| Command | Description |
| --- | --- |
| `pnpm dev:api` / `dev:bot` / `dev:admin` | Run an app in watch mode |
| `pnpm build` | Build all workspace packages |
| `pnpm lint` / `pnpm lint:fix` | Biome check / check --write |
| `pnpm typecheck` | Type-check all workspace packages |
| `pnpm migration:generate` | Generate a TypeORM migration (`packages/database`) |
| `pnpm migration:run` | Run pending migrations |

## Project structure

```
apps/
  api/      NestJS REST API
  bot/      NestJS Telegram bot (Telegraf)
  admin/    Nuxt 3 admin panel
packages/
  database/ TypeORM entities, migrations, data source
```

Backend follows Clean Architecture (entities → use cases → controllers → frameworks/drivers);
the admin frontend follows Feature-Sliced Design. See [CLAUDE.md](CLAUDE.md) for full
conventions.

## Using this as a template

1. Click **Use this template** on GitHub to create a new repository.
2. Rename `parcel-bot-monorepo` in [package.json](package.json) and update this README.
3. Copy `.env.example` to `.env` and fill in real secrets — never commit `.env`.
4. Adjust `apps/*` to fit your domain; `packages/database` holds the shared schema.

## License

[MIT](LICENSE)
