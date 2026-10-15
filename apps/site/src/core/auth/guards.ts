import { AuthStore } from '@/core/auth/auth.store';
import { DEFAULT_LANG, type Lang, localizedPath } from '@/core/i18n/langs';
import { isPlatformServer } from '@angular/common';
import { PLATFORM_ID, inject } from '@angular/core';
import {
  type ActivatedRouteSnapshot,
  type CanActivateFn,
  Router,
  type UrlTree,
} from '@angular/router';

// The language of a route under localizedRoutes (inherited from its parent).
function routeLang(route: ActivatedRouteSnapshot): Lang {
  return (route.data['lang'] as Lang | undefined) ?? DEFAULT_LANG;
}

// Pages for a signed-in visitor (forms, profile): anonymous visitors go to
// /login. These routes render in the browser only (RenderMode.Client).
export const requireAuth: CanActivateFn = async (route): Promise<boolean | UrlTree> => {
  if (isPlatformServer(inject(PLATFORM_ID))) return true;
  const auth = inject(AuthStore);
  const router = inject(Router);
  if (await auth.ensureSession()) return true;
  return router.parseUrl(localizedPath(routeLang(route), '/login'));
};

// Pages for approved partners: everyone else lands on the profile's
// partner tab, where they can apply.
export const requirePartner: CanActivateFn = async (route): Promise<boolean | UrlTree> => {
  if (isPlatformServer(inject(PLATFORM_ID))) return true;
  const auth = inject(AuthStore);
  const router = inject(Router);
  const lang = routeLang(route);
  if (!(await auth.ensureSession())) return router.parseUrl(localizedPath(lang, '/login'));
  if (auth.isPartner()) return true;
  return router.createUrlTree([localizedPath(lang, '/profile')], {
    queryParams: { tab: 'partner' },
  });
};
