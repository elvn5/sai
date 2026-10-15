import type { Hotel } from '@/core/api/types';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { HotelForm, type HotelPayload } from '@/pages/hotels/hotel-form';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /hotels/new: an approved partner adds a hotel (browser only).
@Component({
  selector: 'app-hotel-new-page',
  imports: [HotelForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'hotels.newTitle' | transloco }}</h1>
      <p class="mt-2 text-text-muted">{{ 'hotels.newSubtitle' | transloco }}</p>
      <app-hotel-form class="mt-8 block" [submitLabel]="'listing.form.publish' | transloco" [save]="create" />
    </section>
  `,
})
export class HotelNewPage {
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly create = async (payload: HotelPayload): Promise<void> => {
    const hotel = await firstValueFrom(
      this.http.post<Hotel>('/hotels', { ...payload, slug: slugify(payload.name) }),
    );
    await this.navigate(`/hotels/${hotel.slug}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('hotels.newTitle'),
      description: transloco.translate('hotels.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
