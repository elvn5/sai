import type { Outing } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { OutingCard } from '@/ui/outing-card';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const OUTINGS_LIMIT = 100;

interface DayGroup {
  // Bishkek calendar day, YYYY-MM-DD.
  day: string;
  label: string;
  outings: Outing[];
}

// Upcoming group outings, by day (today, tomorrow, then dates).
@Component({
  selector: 'app-outings-list-page',
  imports: [ListSkeleton, LoadError, OutingCard, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-3xl font-extrabold text-text">{{ 'outings.title' | transloco }}</h1>
          <p class="mt-1 max-w-xl text-text-muted">{{ 'outings.subtitle' | transloco }}</p>
        </div>
        <!-- /outings/new sends visitors who aren't signed in to /login. -->
        <a appSiteLink="/outings/new" class="ui-btn-primary">
          <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'outings.create' | transloco }}
        </a>
      </div>

      <label class="mb-6 inline-flex cursor-pointer items-center gap-2 text-sm text-text">
        <input type="checkbox" class="size-4 accent-brand-600" [checked]="freeOnly()" (change)="freeOnly.set($any($event.target).checked)" />
        {{ 'outings.freeSeatsOnly' | transloco }}
      </label>

      @if (outings.error()) {
        <app-load-error (retry)="outings.reload()" />
      } @else if (groups().length > 0) {
        <div class="flex flex-col gap-8">
          @for (group of groups(); track group.day) {
            <section [attr.aria-labelledby]="'day-' + group.day">
              <h2 [id]="'day-' + group.day" class="mb-3 text-sm font-bold tracking-wide text-text-muted uppercase">{{ group.label }}</h2>
              <ul class="motion-reveal flex flex-col gap-3">
                @for (outing of group.outings; track outing.id) {
                  <li><app-outing-card [outing]="outing" /></li>
                }
              </ul>
            </section>
          }
        </div>
      } @else if (outings.hasValue()) {
        <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
          <span class="icon-[mdi--account-group-outline] size-10 text-text-muted" aria-hidden="true"></span>
          <p class="font-medium text-text">{{ 'outings.empty' | transloco }}</p>
          <p class="max-w-sm text-sm text-text-muted">{{ 'outings.emptyHint' | transloco }}</p>
        </div>
      } @else {
        <app-list-skeleton variant="rows" />
      }
    </section>
  `,
})
export class OutingsListPage {
  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly freeOnly = signal(false);

  protected readonly outings = httpResource<Outing[]>(() => ({
    url: '/outings',
    params: { limit: OUTINGS_LIMIT },
  }));
  private readonly list = computed(() => loaded(this.outings) ?? []);

  protected readonly groups = computed((): DayGroup[] => {
    const today = todayInBishkek();
    const tomorrow = todayInBishkek(new Date(Date.now() + 86_400_000));
    const groups = new Map<string, DayGroup>();
    for (const outing of this.list()) {
      if (this.freeOnly() && outing.taken >= outing.seats) continue;
      const day = todayInBishkek(new Date(outing.startsAt));
      let group = groups.get(day);
      if (!group) {
        const label =
          day === today
            ? this.transloco.translate('outings.groups.today')
            : day === tomorrow
              ? this.transloco.translate('outings.groups.tomorrow')
              : formatDate(this.lang(), day, { weekday: 'long', day: 'numeric', month: 'long' });
        group = { day, label, outings: [] };
        groups.set(day, group);
      }
      group.outings.push(outing);
    }
    return [...groups.values()];
  });

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      if (this.outings.isLoading()) return undefined;
      const heading = this.transloco.translate('outings.title');
      return {
        title: heading,
        description: this.transloco.translate('outings.subtitle'),
        // Outing pages aren't indexed (they're gone in days): no ItemList.
        jsonLd: [seo.breadcrumbs([{ name: heading, path: '/outings' }])],
      };
    });
  }
}
