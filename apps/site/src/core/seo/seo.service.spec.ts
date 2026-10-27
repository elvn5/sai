import { APP_CONFIG, type AppConfig } from '@/core/config/app-config';
import { provideI18n } from '@/core/i18n/i18n';
import { breadcrumbList } from '@/core/seo/json-ld';
import { SeoService } from '@/core/seo/seo.service';
import { DOCUMENT } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

const CONFIG: AppConfig = {
  apiBaseUrl: 'https://api.ermekten.com',
  siteUrl: 'https://ermekten.com',
  telegramBotUsername: '',
  mediaPublicUrl: '',
  indexable: true,
  version: '',
  devMenuPasswordHash: '',
  firebase: null,
};

function setup(
  url: string,
  config: AppConfig = CONFIG,
): { seo: SeoService; head: HTMLHeadElement } {
  TestBed.configureTestingModule({
    providers: [provideRouter([]), provideI18n(), { provide: APP_CONFIG, useValue: config }],
  });
  const router = TestBed.inject(Router);
  Object.defineProperty(router, 'url', { get: () => url });
  return { seo: TestBed.inject(SeoService), head: TestBed.inject(DOCUMENT).head };
}

const attr = (head: HTMLHeadElement, selector: string, name: string): string | null =>
  head.querySelector(selector)?.getAttribute(name) ?? null;

describe('SeoService', () => {
  afterEach(() => {
    for (const element of Array.from(document.head.querySelectorAll('[data-seo], meta'))) {
      element.remove();
    }
  });

  it('writes title, canonical without filters, hreflang and Open Graph', () => {
    const { seo, head } = setup('/en/hotels?sort=price&page=2');
    seo.setPage({
      title: 'Hotels',
      description: 'Where to stay',
      query: { page: 2 },
      image: '/uploads/cover.webp',
    });

    expect(document.title).toBe('Hotels — brand.name');
    expect(attr(head, 'link[rel=canonical]', 'href')).toBe('https://ermekten.com/en/hotels?page=2');
    expect(attr(head, 'link[hreflang="ky"]', 'href')).toBe('https://ermekten.com/ky/hotels?page=2');
    expect(attr(head, 'link[hreflang="x-default"]', 'href')).toBe(
      'https://ermekten.com/hotels?page=2',
    );
    expect(head.querySelectorAll('link[rel=alternate]')).toHaveLength(5);
    expect(attr(head, 'meta[property="og:locale"]', 'content')).toBe('en_US');
    expect(head.querySelectorAll('meta[property="og:locale:alternate"]')).toHaveLength(3);
    expect(attr(head, 'meta[property="og:image"]', 'content')).toBe(
      'https://api.ermekten.com/uploads/cover.webp',
    );
    expect(attr(head, 'meta[name="twitter:card"]', 'content')).toBe('summary_large_image');
    expect(head.querySelector('meta[name=robots]')).toBeNull();
  });

  it('points every prefix of a one-language page at that language', () => {
    const { seo, head } = setup('/en/trails/ala-archa');
    seo.setPage({ title: 'Ala-Archa', description: 'A trail', contentLang: 'ru' });

    expect(attr(head, 'link[rel=canonical]', 'href')).toBe('https://ermekten.com/trails/ala-archa');
    expect(head.querySelectorAll('link[rel=alternate]')).toHaveLength(2);
    expect(attr(head, 'link[hreflang="ru"]', 'href')).toBe('https://ermekten.com/trails/ala-archa');
    expect(attr(head, 'link[hreflang="x-default"]', 'href')).toBe(
      'https://ermekten.com/trails/ala-archa',
    );
    expect(attr(head, 'meta[property="og:locale"]', 'content')).toBe('ru_RU');
    expect(head.querySelectorAll('meta[property="og:locale:alternate"]')).toHaveLength(0);
  });

  it('replaces the previous page instead of piling up tags', () => {
    const { seo, head } = setup('/blog/hello');
    seo.setPage({ title: 'A', description: 'a', image: '/a.webp', jsonLd: [{ '@type': 'A' }] });
    seo.setPage({
      title: 'B',
      description: 'b',
      alternates: null,
      robots: 'noindex',
      jsonLd: [breadcrumbList([{ name: 'Blog', url: 'https://ermekten.com/blog' }])],
    });

    expect(head.querySelectorAll('link[rel=canonical]')).toHaveLength(1);
    expect(head.querySelectorAll('link[rel=alternate]')).toHaveLength(0);
    expect(attr(head, 'meta[property="og:image"]', 'content')).toBe(
      'https://ermekten.com/media/hero-mountain-1920.webp',
    );
    expect(attr(head, 'meta[name=robots]', 'content')).toBe('noindex');
    const scripts = head.querySelectorAll('script[data-seo="page"]');
    expect(scripts).toHaveLength(1);
    expect(JSON.parse(scripts[0]?.textContent ?? '')['@type']).toBe('BreadcrumbList');
  });

  it('keeps staging out of search whatever the page says', () => {
    const { seo, head } = setup('/', { ...CONFIG, indexable: false });
    seo.setPage({ description: 'Home' });
    expect(attr(head, 'meta[name=robots]', 'content')).toBe('noindex, nofollow');
  });

  it('adds Organization and WebSite once, safely escaped', () => {
    const { seo, head } = setup('/');
    seo.setSiteJsonLd('de');
    seo.setSiteJsonLd('de');
    const scripts = Array.from(head.querySelectorAll('script[data-seo="site"]'));
    expect(scripts.map((script) => JSON.parse(script.textContent ?? '')['@type'])).toEqual([
      'Organization',
      'WebSite',
    ]);
    expect(JSON.parse(scripts[1]?.textContent ?? '').inLanguage).toBe('de');
  });
});
