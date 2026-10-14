import { ConfigStore } from '@/shared/config/app-config';
import type { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';

// API calls are written as paths ("/auth/me"): this prefixes the gateway's
// URL and sends the session cookie (admin_token) cross-origin.
export const apiInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith('/')) return next(req);
  const { apiBaseUrl } = inject(ConfigStore).config;
  return next(req.clone({ url: `${apiBaseUrl}${req.url}`, withCredentials: true }));
};
