import { RenderMode, type ServerRoute } from '@angular/ssr';

// Public pages render on the server anonymously, so the same HTML suits
// every visitor: nginx (and browsers) may keep it for 5 minutes and serve
// it stale for 10 more while fetching a fresh copy.
export const PUBLIC_CACHE = 'public, max-age=0, s-maxage=300, stale-while-revalidate=600';

// How each route renders (see docs/angular-migration-plan.md, section 3):
// RenderMode.Server for public catalog and detail pages, RenderMode.Prerender
// for static pages, RenderMode.Client (+ noindex) for account, cabinet,
// payment and edit pages — added with their waves, above the catch-all.
export const serverRoutes: ServerRoute[] = [
  { path: '**', renderMode: RenderMode.Server, headers: { 'Cache-Control': PUBLIC_CACHE } },
];
