import type { Outing } from '@/core/api/types';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { OutingForm, type OutingPayload } from '@/pages/outings/outing-form';
import { TelegramNotice } from '@/pages/outings/telegram-notice';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /outings/new: anyone signed in organizes an outing (browser only);
// ?trail=<slug> comes from a route's "organize" button.
@Component({
  selector: 'app-outing-new-page',
  imports: [OutingForm, TelegramNotice, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-2xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'outings.form.newTitle' | transloco }}</h1>
      <p class="mt-2 text-text-muted">{{ 'outings.form.newHint' | transloco }}</p>
      <app-telegram-notice class="mt-4" />
      <app-outing-form class="mt-8 block" [trailSlug]="trail()" [submitLabel]="'outings.form.create' | transloco" [save]="create" />
    </section>
  `,
})
export class OutingNewPage {
  readonly trail = input<string | undefined>(undefined);
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly create = async (payload: OutingPayload): Promise<void> => {
    const outing = await firstValueFrom(this.http.post<Outing>('/outings', payload));
    await this.navigate(`/outings/${outing.id}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('outings.form.newTitle'),
      description: transloco.translate('outings.subtitle'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
