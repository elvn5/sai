import type { LocalGuide } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { LocalGuideForm, type LocalGuidePayload } from '@/pages/local-guides/local-guide-form';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, effect, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /local-guides/new: a partner creates their guide profile — one per
// account, so someone who has one is sent to edit it.
@Component({
  selector: 'app-local-guide-new-page',
  imports: [LocalGuideForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'localGuides.newTitle' | transloco }}</h1>
      <p class="mt-2 text-text-muted">{{ 'localGuides.newSubtitle' | transloco }}</p>
      @if (mine.hasValue() && !mine.value().guide) {
        <app-local-guide-form class="mt-8 block" [defaultName]="defaultName()" [submitLabel]="'listing.form.publish' | transloco" [save]="create" />
      }
    </section>
  `,
})
export class LocalGuideNewPage {
  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthStore);
  private readonly navigate = injectSiteNavigate();
  private creating = false;

  protected readonly mine = httpResource<{ guide: LocalGuide | null }>(() => '/local-guides/mine');
  protected readonly defaultName = computed(() =>
    [this.auth.user()?.firstName, this.auth.user()?.lastName].filter(Boolean).join(' '),
  );

  protected readonly create = async (payload: LocalGuidePayload): Promise<void> => {
    this.creating = true;
    const guide = await firstValueFrom(
      this.http.post<LocalGuide>('/local-guides', { ...payload, slug: slugify(payload.name) }),
    );
    await this.navigate(`/local-guides/${guide.slug}`);
  };

  constructor() {
    effect(() => {
      const existing = loaded(this.mine)?.guide;
      if (existing && !this.creating) void this.navigate(`/local-guides/${existing.slug}/edit`);
    });
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('localGuides.newTitle'),
      description: transloco.translate('localGuides.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
