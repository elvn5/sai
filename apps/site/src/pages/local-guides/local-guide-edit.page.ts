import type { LocalGuide } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { LocalGuideForm, type LocalGuidePayload } from '@/pages/local-guides/local-guide-form';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /local-guides/:slug/edit: the owner edits their profile.
@Component({
  selector: 'app-local-guide-edit-page',
  imports: [LoadError, LocalGuideForm, NotFoundPage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else {
      <section class="mx-auto max-w-3xl px-4 py-12">
        @if (guide.error()) {
          <app-load-error (retry)="guide.reload()" />
        } @else if (guide.value(); as item) {
          <a [appSiteLink]="'/local-guides/' + item.slug" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ item.name }}</a>
          <h1 class="mt-2 text-3xl font-extrabold text-text">{{ 'localGuides.editTitle' | transloco }}</h1>
          @if (isOwner()) {
            <app-local-guide-form class="mt-8 block" [guide]="item" [submitLabel]="'listing.form.save' | transloco" [save]="update" />
          } @else if (auth.checked()) {
            <p class="mt-8 text-text-muted">{{ 'listing.notOwner' | transloco }}</p>
          }
        }
      </section>
    }
  `,
})
export class LocalGuideEditPage {
  readonly slug = input.required<string>();
  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly navigate = injectSiteNavigate();

  protected readonly guide = httpResource<LocalGuide>(
    () => `/local-guides/${encodeURIComponent(this.slug())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.guide.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && loaded(this.guide)?.userId === user.id);
  });

  protected readonly update = async (payload: LocalGuidePayload): Promise<void> => {
    const id = loaded(this.guide)?.id;
    if (!id) return;
    const updated = await firstValueFrom(this.http.put<LocalGuide>(`/local-guides/${id}`, payload));
    await this.navigate(`/local-guides/${updated.slug}`);
  };

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('localGuides.editTitle'),
      description: transloco.translate('localGuides.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
