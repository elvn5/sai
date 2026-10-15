import { AuthStore } from '@/core/auth/auth.store';
import { isMigrated } from '@/core/config/sections';
import { DEFAULT_LANG, type Lang, localizedPath } from '@/core/i18n/langs';
import { isPlatformServer } from '@angular/common';
import { DOCUMENT, PLATFORM_ID, inject } from '@angular/core';
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

type Redirect = (path: string, query?: Record<string, string>) => boolean | UrlTree;

// Sends the visitor to an unprefixed path in the route's language: inside
// the app when it serves that section, else a full page load (production
// nginx still sends /login and /profile to the Nuxt site). Created before
// the guard awaits, while inject() works.
function injectRedirect(route: ActivatedRouteSnapshot): Redirect {
  const router = inject(Router);
  const location = inject(DOCUMENT).location;
  const lang = routeLang(route);
  return (path, query) => {
    if (isMigrated(path))
      return router.createUrlTree([localizedPath(lang, path)], { queryParams: query });
    const search = query ? `?${new URLSearchParams(query).toString()}` : '';
    location.assign(`${localizedPath(lang, path)}${search}`);
    return false;
  };
}

// Pages for a signed-in visitor (forms, profile): anonymous visitors go to
// /login, which brings them back here. These routes render in the browser
// only (RenderMode.Client).
export const requireAuth: CanActivateFn = async (route, state): Promise<boolean | UrlTree> => {
  if (isPlatformServer(inject(PLATFORM_ID))) return true;
  const auth = inject(AuthStore);
  const redirect = injectRedirect(route);
  return (await auth.ensureSession()) || redirect('/login', { redirect: state.url });
};

// Pages for approved partners: everyone else lands on the profile's
// partner tab, where they can apply.
export const requirePartner: CanActivateFn = async (route, state): Promise<boolean | UrlTree> => {
  if (isPlatformServer(inject(PLATFORM_ID))) return true;
  const auth = inject(AuthStore);
  const redirect = injectRedirect(route);
  if (!(await auth.ensureSession())) return redirect('/login', { redirect: state.url });
  return auth.isPartner() || redirect('/profile', { tab: 'partner' });
};
