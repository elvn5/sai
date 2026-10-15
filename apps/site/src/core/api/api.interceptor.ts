import { APP_CONFIG, SERVER_API_URL } from '@/core/config/app-config';
import { isPlatformServer } from '@angular/common';
import type { HttpInterceptorFn } from '@angular/common/http';
import { PLATFORM_ID, inject } from '@angular/core';

// API calls are written as paths ("/hotels"): this prefixes the gateway's
// URL. In the browser the request carries the session cookie (user_token);
// on the server the page renders anonymously (no cookies, cacheable HTML)
// over the internal API address when there is one — the transfer cache maps
// it back to the public one (app.config.server.ts).
export const apiInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith('/')) return next(req);
  if (isPlatformServer(inject(PLATFORM_ID))) {
    return next(req.clone({ url: `${inject(SERVER_API_URL)}${req.url}` }));
  }
  const { apiBaseUrl } = inject(APP_CONFIG);
  return next(req.clone({ url: `${apiBaseUrl}${req.url}`, withCredentials: true }));
};
