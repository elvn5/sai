import { DEFAULT_LANG, LANGS, type Lang } from '@/core/i18n/langs';
import {
  DOCUMENT,
  type EnvironmentProviders,
  Injectable,
  type Provider,
  inject,
  isDevMode,
} from '@angular/core';
import type { CanActivateFn, Routes } from '@angular/router';
import {
  type Translation,
  type TranslocoLoader,
  TranslocoService,
  provideTransloco,
} from '@jsverse/transloco';
import { type Observable, from, map } from 'rxjs';

// The translations (src/locales). Each language is its own lazy chunk.
const LOADERS: Record<Lang, () => Promise<{ default: Translation }>> = {
  ru: () => import('@/locales/ru.json'),
  ky: () => import('@/locales/ky.json'),
  en: () => import('@/locales/en.json'),
  de: () => import('@/locales/de.json'),
};

@Injectable({ providedIn: 'root' })
export class JsonTranslationLoader implements TranslocoLoader {
  getTranslation(lang: string): Observable<Translation> {
    return from(LOADERS[lang as Lang]().then((module) => module.default));
  }
}

export function provideI18n(): (Provider | EnvironmentProviders)[] {
  return provideTransloco({
    config: {
      availableLangs: [...LANGS],
      defaultLang: DEFAULT_LANG,
      fallbackLang: DEFAULT_LANG,
      missingHandler: { useFallbackTranslation: true, logMissingKey: isDevMode() },
      // vue-i18n's placeholders: "Отзывов: {count}".
      interpolation: ['{', '}'],
      reRenderOnLangChange: true,
      prodMode: !isDevMode(),
    },
    loader: JsonTranslationLoader,
  });
}

// Activates the route's language before it renders, so the server sends
// translated HTML (the router waits for the load) and <html lang> is right.
export const activateLang: CanActivateFn = (route): Observable<boolean> => {
  const lang = route.data['lang'] as Lang;
  const transloco = inject(TranslocoService);
  inject(DOCUMENT).documentElement.lang = lang;
  transloco.setActiveLang(lang);
  return transloco.load(lang).pipe(map(() => true));
};

// Serves `pages` in every language: unprefixed for Russian, under /ky,
// /en and /de for the rest.
export function localizedRoutes(pages: Routes): Routes {
  const prefixed = LANGS.filter((lang) => lang !== DEFAULT_LANG).map((lang) => ({
    path: lang,
    data: { lang },
    canActivate: [activateLang],
    children: pages,
  }));
  return [
    ...prefixed,
    { path: '', data: { lang: DEFAULT_LANG }, canActivate: [activateLang], children: pages },
  ];
}
