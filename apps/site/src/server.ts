import { join } from 'node:path';
import { type RequestContext, configFromEnv } from '@/shared/config/app-config';
import {
  AngularNodeAppEngine,
  createNodeRequestHandler,
  isMainModule,
  writeResponseToNodeResponse,
} from '@angular/ssr/node';
import compression from 'compression';
import express from 'express';

const browserDistFolder = join(import.meta.dirname, '../browser');
// Read once: the environment doesn't change while the server runs.
const config = configFromEnv(process.env);
const context: RequestContext = { config };

const app = express();
app.disable('x-powered-by');
// gzip HTML and assets (nginx may compress too; this keeps direct hits fast).
app.use(compression());
const angularApp = new AngularNodeAppEngine();

app.get('/healthz', (_req, res) => {
  res.json({ status: 'ok' });
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

// Only production is indexable; staging serves the same build.
app.get('/robots.txt', (_req, res) => {
  res.type('text/plain');
  res.send(
    config.indexable
      ? `User-agent: *\nAllow: /\n\nSitemap: ${config.siteUrl}/sitemap.xml\n`
      : 'User-agent: *\nDisallow: /\n',
  );
});

if (!config.indexable) {
  app.use((_req, res, next) => {
    res.setHeader('X-Robots-Tag', 'noindex, nofollow');
    next();
  });
}

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
    process.stdout.write(
      `${JSON.stringify({ level: 'info', msg: 'site listening', port: Number(port) })}\n`,
    );
  });
}

// Used by the Angular CLI dev server and at build time.
export const reqHandler = createNodeRequestHandler(app);
