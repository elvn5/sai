import { BlogEditPage } from '@/pages/blog/blog-edit.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';

describe('BlogEditPage', () => {
  it('derives the slug of a new post from its title and creates it', async () => {
    TestBed.configureTestingModule({ imports: [BlogEditPage], providers: pageTestProviders() });
    const api = TestBed.inject(HttpTestingController);
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const fixture = TestBed.createComponent(BlogEditPage);
    fixture.componentRef.setInput('id', 'new');
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;

    const inputs = el.querySelectorAll<HTMLInputElement>('input.ui-input');
    const [slug, title] = [inputs[0] as HTMLInputElement, inputs[1] as HTMLInputElement];
    title.value = 'Song Kol in Autumn';
    title.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(slug.value).toBe('song-kol-in-autumn');

    el.querySelector('form')?.dispatchEvent(new Event('submit', { cancelable: true }));
    await new Promise((resolve) => setTimeout(resolve));
    const req = api.expectOne('/blog');
    expect(req.request.body).toMatchObject({
      locale: 'ru',
      slug: 'song-kol-in-autumn',
      title: 'Song Kol in Autumn',
      excerpt: null,
      isPublished: false,
    });
    req.flush({ id: 'b1' });
    await new Promise((resolve) => setTimeout(resolve));
    expect(navigate).toHaveBeenCalledWith(['/blog', 'b1']);
  });
});
