import type { Tour } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { TourForm, type TourPayload } from '@/pages/tours/tour-form';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /tours/new: a partner adds a tour; /tours/new?from=<id> copies one of
// theirs for new dates (back to the cabinet afterwards).
@Component({
  selector: 'app-tour-new-page',
  imports: [TourForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ (from() ? 'cabinet.listings.duplicateTitle' : 'tours.newTitle') | transloco }}</h1>
      @if (ready()) {
        <app-tour-form class="mt-8 block" [tour]="source()" [submitLabel]="'tours.submit' | transloco" [save]="create" />
      }
    </section>
  `,
})
export class TourNewPage {
  readonly from = input<string | undefined>(undefined);
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  private readonly mine = httpResource<Tour[]>(() => (this.from() ? '/tours/mine' : undefined));
  protected readonly source = computed((): Partial<Tour> | undefined => {
    const tour = loaded(this.mine)?.find((item) => item.id === this.from());
    return tour ? { ...tour, startDate: '', endDate: '' } : undefined;
  });
  protected readonly ready = computed(() => !this.from() || !this.mine.isLoading());

  protected readonly create = async (payload: TourPayload): Promise<void> => {
    const copy = Boolean(this.from());
    try {
      const tour = await firstValueFrom(
        this.http.post<Tour>('/tours', {
          ...payload,
          locale: this.lang(),
          // A copy shares its title: its dates keep the address unique.
          slug: slugify(copy ? `${payload.title} ${payload.startDate}` : payload.title),
        }),
      );
      await this.navigate(copy ? '/partner/listings' : `/tours/${tour.slug}`);
    } catch (error) {
      throw error instanceof HttpErrorResponse && error.status === 409
        ? 'cabinet.listings.slugTaken'
        : 'tours.createError';
    }
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate(
        this.from() ? 'cabinet.listings.duplicateTitle' : 'tours.newTitle',
      ),
      description: transloco.translate('tours.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
