import type { Tour } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { TourForm, type TourPayload } from '@/pages/tours/tour-form';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/tours/:id: a partner edits one of their tours (their own list
// is the source, so nobody else's can open here).
@Component({
  selector: 'app-partner-tour-edit-page',
  imports: [SiteLink, TourForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex max-w-2xl flex-col gap-2">
      <a appSiteLink="/partner/listings" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ 'cabinet.nav.listings' | transloco }}</a>
      <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.listings.editTour' | transloco }}</h1>
      @if (tour(); as item) {
        <app-tour-form class="mt-6 block" [tour]="item" [submitLabel]="'listing.form.save' | transloco" [save]="update" />
      } @else if (mine.hasValue()) {
        <p class="mt-6 text-text-muted">{{ 'listing.notOwner' | transloco }}</p>
      }
    </div>
  `,
})
export class PartnerTourEditPage {
  readonly id = input.required<string>();
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly mine = httpResource<Tour[]>(() => '/tours/mine');
  protected readonly tour = computed(() =>
    loaded(this.mine)?.find((item) => item.id === this.id()),
  );

  protected readonly update = async (payload: TourPayload): Promise<void> => {
    const id = this.tour()?.id;
    if (!id) return;
    try {
      await firstValueFrom(this.http.put<Tour>(`/tours/${id}`, payload));
    } catch {
      throw 'listing.form.saveError';
    }
    await this.navigate('/partner/listings');
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('cabinet.listings.editTour'),
      description: transloco.translate('tours.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
