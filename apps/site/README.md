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
metric height scale and builds a surrounding elevation mesh from Mapzen Terrarium
(SRTM/GMTED2010, USGS). The route is sampled onto the mesh triangles; GPX data
and the elevation profile remain unchanged. Colors indicate height, not snow.
The viewer loads at most 16 tiles and a 192-cell-long mesh on demand. The fixed
`/terrain/:z/:x/:y.png` proxy bounds coordinates, concurrency and memory, caches
successful tiles and times out upstream requests. Failed loads retain the GPX-only
view with an explicit message and retry. Without terrain, missing GPX elevations
are displayed at the track's minimum known elevation.

Deploy the trails service with migration `00007_trail_posts.sql` (applied on service
startup) for markers and PDF checklist URLs. The media service supports
`?download=1` for attachment downloads, including when the API uses another origin.

Angular Three 4.2.4 declares Angular peers below 22; this project uses Angular 22.
Production compilation and a Chrome WebGL 2 editor/save flow have been verified,
but recheck this integration on Angular upgrades. Three.js is kept on 0.182.x,
within Angular Three's declared supported range. Without WebGL 2, the canvas and
height profile remain usable.
