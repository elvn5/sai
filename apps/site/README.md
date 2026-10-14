# apps/site

The public Ermekten site and Telegram Mini App on Angular 22 with SSR —
the successor of `apps/tourism` (Nuxt). Plan and SEO requirements:
`docs/angular-migration-plan.md`.

- `pnpm dev:site` — dev server on http://localhost:3012 (`nvm use`: Node from `.nvmrc`)
- `pnpm --filter @app/site test | lint | build`
- `pnpm --filter @app/site start` — the built SSR server (`PORT`, `API_BASE_URL`,
  `SITE_URL`, `TELEGRAM_BOT_USERNAME` from the environment)
- Lighthouse budgets: `lighthouserc.json` (CI job `lighthouse`)
