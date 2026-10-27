import type { Trail } from '@/core/api/types';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { TrailForm, type TrailPayload } from '@/pages/trails/trail-form';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /trails/new: an approved partner publishes a route (browser only).
@Component({
  selector: 'app-trail-new-page',
  imports: [TrailForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'trails.form.newTitle' | transloco }}</h1>
      <app-trail-form class="mt-8 block" [submitLabel]="'trails.form.create' | transloco" [save]="create" />
    </section>
  `,
})
export class TrailNewPage {
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly create = async (payload: TrailPayload): Promise<void> => {
    const trail = await firstValueFrom(
      this.http.post<Trail>('/trails', { ...payload, slug: slugify(payload.name) }),
    );
    await this.navigate(`/trails/${trail.slug}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('trails.form.newTitle'),
      description: transloco.translate('trails.subtitle'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
