import { LANGS, localizedPath } from '@/core/i18n/langs';
import { RenderMode, type ServerRoute } from '@angular/ssr';

// Public pages render on the server anonymously, so the same HTML suits
// every visitor: nginx (and browsers) may keep it for 5 minutes and serve
// it stale for 10 more while fetching a fresh copy.
export const PUBLIC_CACHE = 'public, max-age=0, s-maxage=300, stale-while-revalidate=600';

// Account, cabinet, payment and edit pages: rendered in the browser only
// (never cached, nothing personal on the server). Unprefixed paths.
const CLIENT_PATHS = ['blog/new', 'blog/edit/:id', 'guides/new'];

// The same route under every language prefix ("blog/new", "en/blog/new"…).
function inEveryLanguage(path: string): string[] {
  return LANGS.map((lang) => localizedPath(lang, `/${path}`).slice(1));
}

// How each route renders (see docs/angular-migration-plan.md, section 3):
// RenderMode.Server for public catalog and detail pages (the catch-all),
// RenderMode.Prerender for static pages, RenderMode.Client (+ noindex) for
// account, cabinet, payment and edit pages.
export const serverRoutes: ServerRoute[] = [
  ...CLIENT_PATHS.flatMap(inEveryLanguage).map(
    (path): ServerRoute => ({ path, renderMode: RenderMode.Client }),
  ),
  { path: '**', renderMode: RenderMode.Server, headers: { 'Cache-Control': PUBLIC_CACHE } },
];
