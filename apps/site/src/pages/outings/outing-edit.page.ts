import type { Outing } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { OutingForm, type OutingPayload } from '@/pages/outings/outing-form';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /outings/:id/edit: the organizer changes their outing (browser only;
// the request carries the session, so isOrganizer is the visitor's).
@Component({
  selector: 'app-outing-edit-page',
  imports: [LoadError, NotFoundPage, OutingForm, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else {
      <section class="mx-auto max-w-2xl px-4 py-12">
        @if (outing.error()) {
          <app-load-error (retry)="outing.reload()" />
        } @else if (outing.value(); as item) {
          <a [appSiteLink]="'/outings/' + item.id" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ item.title }}</a>
          <h1 class="mt-2 text-3xl font-extrabold text-text">{{ 'outings.form.editTitle' | transloco }}</h1>
          @if (item.isOrganizer) {
            <app-outing-form class="mt-8 block" [outing]="item" [submitLabel]="'outings.form.save' | transloco" [save]="update" />
          } @else {
            <p class="mt-8 text-text-muted">{{ 'outings.notYours' | transloco }}</p>
          }
        }
      </section>
    }
  `,
})
export class OutingEditPage {
  readonly id = input.required<string>();
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly outing = httpResource<Outing>(
    () => `/outings/${encodeURIComponent(this.id())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.outing.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });

  protected readonly update = async (payload: OutingPayload): Promise<void> => {
    const id = loaded(this.outing)?.id;
    if (!id) return;
    await firstValueFrom(this.http.put<Outing>(`/outings/${id}`, payload));
    await this.navigate(`/outings/${id}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('outings.form.editTitle'),
      description: transloco.translate('outings.subtitle'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
