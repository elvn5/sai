import type { Hotel } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { HotelForm, type HotelPayload } from '@/pages/hotels/hotel-form';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /hotels/:slug/edit: the owner edits their hotel (browser only).
@Component({
  selector: 'app-hotel-edit-page',
  imports: [HotelForm, LoadError, NotFoundPage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else {
      <section class="mx-auto max-w-3xl px-4 py-12">
        @if (hotel.error()) {
          <app-load-error (retry)="hotel.reload()" />
        } @else if (hotel.value(); as item) {
          <a [appSiteLink]="'/hotels/' + item.slug" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ item.name }}</a>
          <h1 class="mt-2 text-3xl font-extrabold text-text">{{ 'hotels.editTitle' | transloco }}</h1>
          @if (isOwner()) {
            <app-hotel-form class="mt-8 block" [hotel]="item" [submitLabel]="'listing.form.save' | transloco" [save]="update" />
          } @else if (auth.checked()) {
            <p class="mt-8 text-text-muted">{{ 'listing.notOwner' | transloco }}</p>
          }
        }
      </section>
    }
  `,
})
export class HotelEditPage {
  readonly slug = input.required<string>();
  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly hotel = httpResource<Hotel>(
    () => `/hotels/${encodeURIComponent(this.slug())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.hotel.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && loaded(this.hotel)?.userId === user.id);
  });

  protected readonly update = async (payload: HotelPayload): Promise<void> => {
    const id = loaded(this.hotel)?.id;
    if (!id) return;
    const updated = await firstValueFrom(this.http.put<Hotel>(`/hotels/${id}`, payload));
    await this.navigate(`/hotels/${updated.slug}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('hotels.editTitle'),
      description: transloco.translate('hotels.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
