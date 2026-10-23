import type { BlogPost } from '@/core/api/types';
import { BlogPostPage } from '@/pages/blog/blog-post.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { DOCUMENT } from '@angular/core';
import { TestBed } from '@angular/core/testing';

const POST: BlogPost = {
  id: 'p1',
  locale: 'ru',
  slug: 'son-kul',
  title: 'Сон-Куль летом',
  excerpt: 'Озеро на высоте 3000 м',
  content: '<p>Текст <script>alert(1)</script></p>',
  coverImageUrl: '/uploads/son-kul.webp',
  seoTitle: null,
  seoDescription: null,
  difficulty: 'easy',
  duration: '1 день',
  season: 'summer',
  isPublished: true,
  publishedAt: '2026-09-01T08:00:00Z',
  userId: null,
  updatedAt: '2026-09-02T08:00:00Z',
};

async function render(response: ResponseInit = {}) {
  TestBed.configureTestingModule({ providers: pageTestProviders(response) });
  const fixture = TestBed.createComponent(BlogPostPage);
  fixture.componentRef.setInput('slug', 'son-kul');
  fixture.detectChanges();
  return { fixture, http: TestBed.inject(HttpTestingController) };
}

describe('BlogPostPage', () => {
  afterEach(() => {
    for (const element of Array.from(document.head.querySelectorAll('[data-seo], meta')))
      element.remove();
  });

  it('renders the post with BlogPosting and breadcrumbs, without hreflang', async () => {
    const { fixture, http } = await render();
    http.expectOne('/blog/ru/son-kul').flush(POST);
    await fixture.whenStable();

    const element: HTMLElement = fixture.nativeElement;
    expect(element.querySelector('h1')?.textContent).toContain('Сон-Куль летом');
    // Angular's sanitizer drops scripts from the post HTML.
    expect(element.querySelector('.prose')?.innerHTML).not.toContain('<script');
    // Difficulty, duration and season as badges.
    const badges = Array.from(element.querySelectorAll('app-post-badges li'));
    expect(badges).toHaveLength(3);
    expect(badges[1].textContent).toContain('1 день');

    const head = TestBed.inject(DOCUMENT).head;
    const types = Array.from(head.querySelectorAll('script[data-seo="page"]')).map(
      (script) => JSON.parse(script.textContent ?? '')['@type'],
    );
    expect(types).toEqual(['BlogPosting', 'BreadcrumbList']);
    expect(head.querySelectorAll('link[hreflang]')).toHaveLength(0);
    expect(head.querySelector('meta[property="og:type"]')?.getAttribute('content')).toBe('article');
    expect(head.querySelector('meta[property="og:image"]')?.getAttribute('content')).toBe(
      'https://api.test/uploads/son-kul.webp',
    );
  });

  it('answers 404 for an unknown slug', async () => {
    const response: ResponseInit = {};
    const { fixture, http } = await render(response);
    http.expectOne('/blog/ru/son-kul').flush(null, { status: 404, statusText: 'Not Found' });
    await fixture.whenStable();
    expect(response.status).toBe(404);
    expect(fixture.nativeElement.querySelector('app-not-found-page')).not.toBeNull();
  });

  it('answers 503 when the API fails', async () => {
    const response: ResponseInit = {};
    const { fixture, http } = await render(response);
    http.expectOne('/blog/ru/son-kul').flush(null, { status: 502, statusText: 'Bad Gateway' });
    await fixture.whenStable();
    expect(response.status).toBe(503);
    expect(fixture.nativeElement.querySelector('app-load-error')).not.toBeNull();
  });
});
