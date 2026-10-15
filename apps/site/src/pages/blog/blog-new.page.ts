import { bindPageMeta } from '@/core/seo/seo.service';
import { BlogForm } from '@/pages/blog/blog-form';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// /blog/new: an approved partner writes a post (rendered in the browser only).
@Component({
  selector: 'app-blog-new-page',
  imports: [BlogForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'blog.form.newTitle' | transloco }}</h1>
      <app-blog-form />
    </section>
  `,
})
export class BlogNewPage {
  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('blog.form.newTitle'),
      description: transloco.translate('blog.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
