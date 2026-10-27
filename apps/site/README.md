# apps/site

The public Ermekten site and Telegram Mini App on Angular 22 with SSR. SEO
requirements: CLAUDE.md (Angular Apps) and `docs/angular-migration-plan.md`, section 4.

- `pnpm dev:site` — dev server on http://localhost:3012 (`nvm use`: Node from `.nvmrc`)
- `pnpm --filter @app/site test | lint | build`
- `pnpm --filter @app/site start` — the built SSR server (`PORT`, `API_BASE_URL`,
  `SITE_URL`, `TELEGRAM_BOT_USERNAME` from the environment)
- Lighthouse budgets: `lighthouserc.json` (CI job `lighthouse`)

## Route posts

`/trails` is the public catalog; approved partners create posts at `/trails/new`
and edit their own at `/trails/:slug/edit`. GPX imports feed the canvas, elevation
profile and lazy Angular Three viewer. Markers reference track point indices;
replacing a track clears old markers. Automatic suggestions flag at least 20%
grade over at least 100 m and require the author's review. The 3D viewer preserves
metric height scale and displays the track, not a terrain elevation mesh.
Missing elevations are displayed at the track's minimum known elevation.

Deploy the trails service with migration `00006_trail_posts.sql` (applied on service
startup) for markers and PDF checklist URLs. The media service supports
`?download=1` for attachment downloads, including when the API uses another origin.

Angular Three 4.2.4 declares Angular peers below 22; this project uses Angular 22.
Production compilation and a Chrome WebGL 2 editor/save flow have been verified,
but recheck this integration on Angular upgrades. Three.js is kept on 0.182.x,
within Angular Three's declared supported range. Without WebGL 2, the canvas and
height profile remain usable.
