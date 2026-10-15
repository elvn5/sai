import type { Trail } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { TrailForm, type TrailPayload } from '@/pages/trails/trail-form';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /trails/:slug/edit: the author edits their route (browser only; a draft
// is readable here because the request carries the session).
@Component({
  selector: 'app-trail-edit-page',
  imports: [LoadError, NotFoundPage, SiteLink, TrailForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else {
      <section class="mx-auto max-w-3xl px-4 py-12">
        @if (trail.error()) {
          <app-load-error (retry)="trail.reload()" />
        } @else if (trail.value(); as item) {
          <a [appSiteLink]="'/trails/' + item.slug" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ item.name }}</a>
          <h1 class="mt-2 text-3xl font-extrabold text-text">{{ 'trails.form.editTitle' | transloco }}</h1>
          @if (isOwner()) {
            <app-trail-form class="mt-8 block" [trail]="item" [submitLabel]="'trails.form.save' | transloco" [save]="update" />
          } @else if (auth.checked()) {
            <p class="mt-8 text-text-muted">{{ 'trails.notYours' | transloco }}</p>
          }
        }
      </section>
    }
  `,
})
export class TrailEditPage {
  readonly slug = input.required<string>();
  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly trail = httpResource<Trail>(
    () => `/trails/${encodeURIComponent(this.slug())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.trail.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && loaded(this.trail)?.userId === user.id);
  });

  protected readonly update = async (payload: TrailPayload): Promise<void> => {
    const id = loaded(this.trail)?.id;
    if (!id) return;
    const updated = await firstValueFrom(this.http.put<Trail>(`/trails/${id}`, payload));
    await this.navigate(`/trails/${updated.slug}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('trails.form.editTitle'),
      description: transloco.translate('trails.subtitle'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
