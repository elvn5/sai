# apps/panel

The Ermekten admin panel on Angular 22 (client-only) — the successor of
`apps/admin` (Nuxt). Plan: `docs/angular-migration-plan.md`.

- `pnpm dev:panel` — dev server on http://localhost:3013 (`nvm use`: Node from `.nvmrc`);
  `public/config.json` points it at the local gateway
- `pnpm --filter @app/panel test | lint | build`
- In Docker, nginx serves the build and writes `config.json` from
  `API_BASE_URL` on start (`docker-entrypoint.sh`)
