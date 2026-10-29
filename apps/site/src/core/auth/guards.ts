import { AuthStore } from '@/core/auth/auth.store';
import { DEFAULT_LANG, type Lang, localizedPath } from '@/core/i18n/langs';
import { isPlatformServer } from '@angular/common';
import { PLATFORM_ID, inject } from '@angular/core';
import {
  type ActivatedRouteSnapshot,
  type CanActivateFn,
  type CanMatchFn,
  Router,
  type UrlTree,
} from '@angular/router';

// The language of a route under localizedRoutes (inherited from its parent).
function routeLang(route: ActivatedRouteSnapshot): Lang {
  return (route.data['lang'] as Lang | undefined) ?? DEFAULT_LANG;
}

type Redirect = (path: string, query?: Record<string, string>) => UrlTree;

// Sends the visitor to an unprefixed path in the route's language. Created
// before the guard awaits, while inject() works.
function injectRedirect(route: ActivatedRouteSnapshot): Redirect {
  const router = inject(Router);
  const lang = routeLang(route);
  return (path, query) => router.createUrlTree([localizedPath(lang, path)], { queryParams: query });
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

// Matches the partner cabinet's driver pages (and the driver's dashboard
// at /partner) for drivers only; organizers fall through to their own.
export const matchDriver: CanMatchFn = async (): Promise<boolean> => {
  if (isPlatformServer(inject(PLATFORM_ID))) return false;
  const auth = inject(AuthStore);
  return (await auth.ensureSession()) && auth.isDriver();
};
