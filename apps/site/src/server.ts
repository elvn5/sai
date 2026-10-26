import { join } from 'node:path';
import { contextFromEnv } from '@/core/config/app-config';
import { movedSectionRedirect, pausedSectionRedirect } from '@/server/paused-sections';
import { SitemapCache, isSitemapType, renderIndex } from '@/server/sitemap';
import {
  AngularNodeAppEngine,
  createNodeRequestHandler,
  isMainModule,
  writeResponseToNodeResponse,
} from '@angular/ssr/node';
import compression from 'compression';
import express from 'express';
import { version } from '#package.json';

const browserDistFolder = join(import.meta.dirname, '../browser');
// Read once: the environment doesn't change while the server runs.
// The version defaults to package.json's; APP_VERSION (a release tag) wins.
const context = contextFromEnv({ APP_VERSION: version, ...process.env });
const { config } = context;
const apiUrl = context.apiInternalUrl ?? config.apiBaseUrl;
const sitemaps = new SitemapCache(config.siteUrl, async (path) => {
  const response = await fetch(`${apiUrl}${path}`, { signal: AbortSignal.timeout(10_000) });
  if (!response.ok) throw new Error(`GET ${path}: ${response.status}`);
  return response.json();
});
const SITEMAP_CACHE = 'public, max-age=600';

const app = express();
app.disable('x-powered-by');
// gzip HTML and assets (nginx may compress too; this keeps direct hits fast).
app.use(compression());
const angularApp = new AngularNodeAppEngine();

app.get('/healthz', (_req, res) => {
  res.json({ status: 'ok' });
});

// One host: www.ermekten.com → ermekten.com (301).
app.use((req, res, next) => {
  if (req.hostname.startsWith('www.')) {
    res.redirect(301, `${config.siteUrl}${req.originalUrl}`);
    return;
  }
  next();
});

// One URL per page: /tours/ → /tours (301), keeping the query string.
app.use((req, res, next) => {
  const [path, query] = req.originalUrl.split(/\?(.*)/s, 2);
  if (path && path.length > 1 && path.endsWith('/')) {
    res.redirect(301, path.replace(/\/+$/, '') + (query ? `?${query}` : ''));
    return;
  }
  next();
});

// Sections paused for now: their addresses lead to what's left (302);
// removed ones moved for good (301).
app.use((req, res, next) => {
  const moved = movedSectionRedirect(req.originalUrl);
  if (moved) {
    res.redirect(301, moved);
    return;
  }
  const target = pausedSectionRedirect(req.originalUrl);
  if (target) {
    res.redirect(302, target);
    return;
  }
  next();
});

// Only production is indexable; staging serves the same build.
app.get('/robots.txt', (_req, res) => {
  res.type('text/plain');
  res.send(
    config.indexable
      ? `User-agent: *\nAllow: /\n\nSitemap: ${config.siteUrl}/sitemap.xml\n`
      : 'User-agent: *\nDisallow: /\n',
  );
});

app.get('/sitemap.xml', (_req, res) => {
  res
    .type('application/xml')
    .setHeader('Cache-Control', SITEMAP_CACHE)
    .send(renderIndex(config.siteUrl));
});

app.get('/sitemaps/:file', (req, res, next) => {
  const type = req.params.file.replace(/\.xml$/, '');
  if (!req.params.file.endsWith('.xml') || !isSitemapType(type)) {
    next();
    return;
  }
  sitemaps.get(type).then(
    (xml) => res.type('application/xml').setHeader('Cache-Control', SITEMAP_CACHE).send(xml),
    (error: unknown) => {
      // Crawlers retry a 503 later instead of dropping the URLs.
      log('error', 'sitemap failed', { type, error: String(error) });
      res
        .status(503)
        .setHeader('Retry-After', '300')
        .type('text/plain')
        .send('Sitemap unavailable');
    },
  );
});

if (!config.indexable) {
  app.use((_req, res, next) => {
    res.setHeader('X-Robots-Tag', 'noindex, nofollow');
    next();
  });
}

// The runtime config for pages rendered in the browser only (no server
// state to carry it): the same public values the HTML would hold.
app.get('/app-config.json', (_req, res) => {
  res.setHeader('Cache-Control', 'no-store');
  res.json(config);
});

// Hashed build assets: cache forever.
app.use(express.static(browserDistFolder, { maxAge: '1y', index: false, redirect: false }));

// Everything else renders the Angular app on the server.
app.use((req, res, next) => {
  angularApp
    .handle(req, context)
    .then((response) => (response ? writeResponseToNodeResponse(response, res) : next()))
    .catch(next);
});

if (isMainModule(import.meta.url) || process.env['pm_id']) {
  const port = process.env['PORT'] || 4000;
  app.listen(port, (error) => {
    if (error) throw error;
    log('info', 'site listening', { port: Number(port) });
  });
}

// JSON lines on stdout, like the Go services' slog.
function log(level: 'info' | 'error', msg: string, fields: Record<string, unknown> = {}): void {
  process.stdout.write(`${JSON.stringify({ level, msg, ...fields })}\n`);
}

// Used by the Angular CLI dev server and at build time.
export const reqHandler = createNodeRequestHandler(app);
