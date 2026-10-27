import type { PageSeo, SeoPage } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { LANGS, type Lang, OG_LOCALES, localizedPath, splitLang } from '@/core/i18n/langs';
import { resolveMediaUrl } from '@/core/lib/media-url';
import {
  type JsonLd,
  breadcrumbList,
  organization,
  serializeJsonLd,
  webSite,
} from '@/core/seo/json-ld';
import { type HttpResourceRef, httpResource } from '@angular/common/http';
import { DOCUMENT, Injectable, effect, inject, untracked } from '@angular/core';
import { Meta, Title } from '@angular/platform-browser';
import { Router } from '@angular/router';
import { TranslocoService } from '@jsverse/transloco';

// What a page tells search engines and link previews about itself.
export interface PageMeta {
  // The page's own title; the site name is appended. None on the home page.
  title?: string;
  description: string;
  // Preview image (absolute, or an /uploads path of the API), ideally
  // 1200×630; the site's hero photo when there is none.
  image?: string | null;
  type?: 'website' | 'article' | 'profile';
  // The page's path in each language. By default it is the same path in
  // every language (hotels, trails); null for content that exists in one
  // language only (a blog post has its own slug per language).
  alternates?: Partial<Record<Lang, string>> | null;
  // The language the text is written in, when it is the same text under
  // every prefix (trails, legal documents): the canonical URL and the only
  // hreflang are that language's address, so the other prefixes don't pass
  // for translations.
  contentLang?: Lang;
  // Query parameters that make a different page (?page=2): kept in the
  // canonical URL. Filters and sorting are not, so they are never passed.
  query?: Record<string, string | number>;
  // "noindex, follow" for filtered lists, "noindex" for private pages.
  robots?: string;
  // The page's schema.org data (BreadcrumbList, TouristTrip…).
  jsonLd?: JsonLd[];
}

const PAGE_MARK = 'data-seo';
// The brand is the same in every language, and site-wide JSON-LD is set
// before a page's translations have loaded.
const BRAND = 'Ermekten';
// The preview image of pages without one of their own (public/media).
const DEFAULT_IMAGE = '/media/hero-mountain-1920.webp';

// Writes <head> for the current page: title, description, canonical,
// hreflang, Open Graph, Twitter Card, robots and JSON-LD. Pages call
// setPage() once their data is known (see docs/angular-migration-plan.md,
// section 4); on the server this lands in the rendered HTML.
@Injectable({ providedIn: 'root' })
export class SeoService {
  private readonly document = inject(DOCUMENT);
  private readonly title = inject(Title);
  private readonly meta = inject(Meta);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);
  private readonly config = inject(APP_CONFIG);

  setPage(page: PageMeta): void {
    const { lang, path } = splitLang(this.currentUrl());
    const brand = this.transloco.translate('brand.name');
    const title = page.title
      ? `${page.title} — ${brand}`
      : `${brand} — ${this.transloco.translate('home.metaTitle')}`;
    const pageLang = page.contentLang ?? lang;
    const canonical = this.absolute(localizedPath(pageLang, path), page.query);

    this.title.setTitle(title);
    this.setName('description', page.description);
    this.setRobots(page.robots);

    this.clearLinks();
    this.addLink({ rel: 'canonical', href: canonical });
    const alternates = page.contentLang
      ? ([[page.contentLang, path]] as [Lang, string][])
      : this.alternates(path, page.alternates);
    for (const [alternateLang, alternatePath] of alternates) {
      this.addLink({
        rel: 'alternate',
        hreflang: alternateLang,
        href: this.absolute(localizedPath(alternateLang, alternatePath), page.query),
      });
    }
    const russian = alternates.find(([alternateLang]) => alternateLang === 'ru');
    if (russian) {
      this.addLink({
        rel: 'alternate',
        hreflang: 'x-default',
        href: this.absolute(russian[1], page.query),
      });
    }

    const image = page.image
      ? resolveMediaUrl(this.config.apiBaseUrl, page.image)
      : `${this.config.siteUrl}${DEFAULT_IMAGE}`;
    this.setProperty('og:type', page.type ?? 'website');
    this.setProperty('og:title', page.title ?? title);
    this.setProperty('og:description', page.description);
    this.setProperty('og:url', canonical);
    this.setProperty('og:locale', OG_LOCALES[pageLang]);
    for (const tag of this.meta.getTags('property="og:locale:alternate"')) {
      this.meta.removeTagElement(tag);
    }
    for (const [alternateLang] of alternates) {
      if (alternateLang !== pageLang) {
        this.meta.addTag({ property: 'og:locale:alternate', content: OG_LOCALES[alternateLang] });
      }
    }
    this.setProperty('og:image', image);
    // A profile's picture is a small square avatar.
    this.setName('twitter:card', page.type === 'profile' ? 'summary' : 'summary_large_image');
    this.setName('twitter:title', page.title ?? title);
    this.setName('twitter:description', page.description);
    this.setName('twitter:image', image);

    this.setJsonLd('page', page.jsonLd ?? []);
  }

  // Site-wide JSON-LD (Organization, WebSite), set once by the root component.
  setSiteJsonLd(lang: Lang): void {
    this.setJsonLd('site', [
      organization(this.config.siteUrl, BRAND),
      webSite(this.config.siteUrl, BRAND, lang),
    ]);
  }

  // The absolute URL of an unprefixed path ("/blog/hello") in the current
  // language.
  pageUrl(path: string): string {
    return this.absolute(localizedPath(splitLang(this.currentUrl()).lang, path));
  }

  // The current language's code (inLanguage).
  langTag(): string {
    return splitLang(this.currentUrl()).lang;
  }

  // BreadcrumbList from the home page through `crumbs` (unprefixed paths;
  // the last one is the page itself).
  breadcrumbs(crumbs: { name: string; path: string }[]): JsonLd {
    const home = { name: this.transloco.translate('breadcrumbs.home'), path: '/' };
    return breadcrumbList(
      [home, ...crumbs].map(({ name, path }) => ({ name, url: this.pageUrl(path) })),
    );
  }

  // The absolute URL of a path that already carries its language prefix.
  absolute(path: string, query?: Record<string, string | number>): string {
    const search = query
      ? new URLSearchParams(
          Object.entries(query).map(([key, value]) => [key, String(value)]),
        ).toString()
      : '';
    return `${this.config.siteUrl}${path}${search ? `?${search}` : ''}`;
  }

  // While a navigation is running (pages call setPage from their
  // constructors) router.url is still the previous page.
  private currentUrl(): string {
    const navigation = this.router.currentNavigation();
    return navigation?.finalUrl ? this.router.serializeUrl(navigation.finalUrl) : this.router.url;
  }

  private alternates(path: string, alternates: PageMeta['alternates']): [Lang, string][] {
    if (alternates === null) return [];
    if (alternates === undefined) return LANGS.map((lang) => [lang, path]);
    return LANGS.flatMap((lang): [Lang, string][] => {
      const alternatePath = alternates[lang];
      return alternatePath ? [[lang, alternatePath]] : [];
    });
  }

  private setRobots(robots: string | undefined): void {
    // Staging and previews serve the same build: keep them out of search.
    const content = this.config.indexable ? robots : 'noindex, nofollow';
    this.setName('robots', content ?? null);
  }

  private setName(name: string, content: string | null): void {
    if (content) this.meta.updateTag({ name, content });
    else this.meta.removeTag(`name="${name}"`);
  }

  private setProperty(property: string, content: string | null): void {
    if (content) this.meta.updateTag({ property, content });
    else this.meta.removeTag(`property="${property}"`);
  }

  private clearLinks(): void {
    for (const link of Array.from(this.document.head.querySelectorAll(`link[${PAGE_MARK}]`))) {
      link.remove();
    }
  }

  private addLink(attributes: Record<string, string>): void {
    const link = this.document.createElement('link');
    link.setAttribute(PAGE_MARK, '');
    for (const [name, value] of Object.entries(attributes)) link.setAttribute(name, value);
    this.document.head.appendChild(link);
  }

  private setJsonLd(scope: 'site' | 'page', items: JsonLd[]): void {
    const selector = `script[type="application/ld+json"][${PAGE_MARK}="${scope}"]`;
    for (const script of Array.from(this.document.head.querySelectorAll(selector))) script.remove();
    for (const item of items) {
      const script = this.document.createElement('script');
      script.setAttribute('type', 'application/ld+json');
      script.setAttribute(PAGE_MARK, scope);
      script.textContent = serializeJsonLd(item);
      this.document.head.appendChild(script);
    }
  }
}

// The admin's overrides (/page-seo) of a catalog page's title, description
// and preview image in the current language; null when there are none.
export function pageSeoResource(page: SeoPage): HttpResourceRef<PageSeo | null | undefined> {
  const lang = injectLang();
  return httpResource<PageSeo | null>(() => `/page-seo/${page}/${lang()}`);
}

// Keeps <head> in step with `meta` (re-run when the signals it reads
// change: the page's data, the overrides, the language). Call from a page's
// constructor.
export function bindPageMeta(meta: () => PageMeta | undefined): void {
  const seo = inject(SeoService);
  effect(() => {
    const page = meta();
    // Only `meta` is tracked: setPage reads the router's navigation signal,
    // which must not re-run this for the next page while leaving.
    if (page) untracked(() => seo.setPage(page));
  });
}
