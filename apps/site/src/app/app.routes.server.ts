import { LANGS, localizedPath } from '@/core/i18n/langs';
import { RenderMode, type ServerRoute } from '@angular/ssr';

// Public pages render on the server anonymously, so the same HTML suits
// every visitor: nginx (and browsers) may keep it for 5 minutes and serve
// it stale for 10 more while fetching a fresh copy.
export const PUBLIC_CACHE = 'public, max-age=0, s-maxage=300, stale-while-revalidate=600';

// The page's own noindex arrives with its script; crawlers that don't run
// it still read this header (the shell they get is the same everywhere).
const PRIVATE_HEADERS = { 'X-Robots-Tag': 'noindex', 'Cache-Control': 'no-store' };

// Account, cabinet, payment and edit pages: rendered in the browser only
// (never cached, nothing personal on the server). Unprefixed paths.
const CLIENT_PATHS = [
  'blog/new',
  'blog/edit/:id',
  'guides/new',
  'hotels/new',
  'hotels/:slug/edit',
  'local-guides/new',
  'local-guides/:slug/edit',
  'tours/new',
  'trails/new',
  'trails/:slug/edit',
  'outings/new',
  'outings/:id/edit',
  'login',
  'profile',
  'payments/return',
  'payments/test',
  'achievements/award',
  'partner',
  'partner/**',
];

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
    (path): ServerRoute => ({ path, renderMode: RenderMode.Client, headers: PRIVATE_HEADERS }),
  ),
  { path: '**', renderMode: RenderMode.Server, headers: { 'Cache-Control': PUBLIC_CACHE } },
];
