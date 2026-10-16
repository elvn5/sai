import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { inject } from '@angular/core';
import { Router } from '@angular/router';

// Navigates to an unprefixed site path ("/blog/hello") in the current
// language. See SiteLink.
export function injectSiteNavigate(): (path: string) => Promise<void> {
  const router = inject(Router);
  const lang = injectLang();
  return async (path) => {
    await router.navigateByUrl(localizedPath(lang(), path));
  };
}
