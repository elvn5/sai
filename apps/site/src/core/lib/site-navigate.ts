import { isMigrated } from '@/core/config/sections';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { DOCUMENT, inject } from '@angular/core';
import { Router } from '@angular/router';

// Navigates to an unprefixed site path ("/blog/hello") in the current
// language: inside the app for sections it serves, else a full page load
// (production nginx still sends the rest to Nuxt). See SiteLink.
export function injectSiteNavigate(): (path: string) => Promise<void> {
  const router = inject(Router);
  const lang = injectLang();
  const location = inject(DOCUMENT).location;
  return async (path) => {
    const url = localizedPath(lang(), path);
    if (isMigrated(path.split('?')[0] ?? path)) await router.navigateByUrl(url);
    else location.assign(url);
  };
}
