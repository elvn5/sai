import type { StatsReport } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import {
  contentLang,
  downloadCsv,
  listingPath,
  percentChange,
  shareOf,
} from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { DailyBarChart } from '@/pages/partner/daily-bar-chart';
import { StatTile } from '@/pages/partner/stat-tile';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  computed,
  inject,
  linkedSignal,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const PERIODS = [7, 30, 90] as const;

// /partner/stats: views and contact clicks per period, per channel and per
// listing, with a CSV export.
@Component({
  selector: 'app-partner-stats-page',
  imports: [DailyBarChart, SiteLink, StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.nav.stats' | transloco }}</h1>
          <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.stats.subtitle' | transloco }}</p>
        </div>
        <div class="flex items-center gap-2">
          <div class="flex rounded-lg border border-border bg-surface p-0.5" role="group" [attr.aria-label]="'cabinet.nav.stats' | transloco">
            @for (period of periods; track period) {
              <button type="button" class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors" [class]="days() === period ? 'bg-brand-600 text-white' : 'text-text-muted hover:text-text'" [attr.aria-pressed]="days() === period" (click)="days.set(period)">
                {{ 'cabinet.stats.days' | transloco: { count: period } }}
              </button>
            }
          </div>
          <button type="button" class="ui-btn-soft" [disabled]="!report()" (click)="exportCsv()">
            <span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>CSV
          </button>
        </div>
      </div>

      @if (report(); as r) {
        <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <app-stat-tile [label]="'cabinet.stats.views' | transloco" icon="icon-[heroicons--eye]" [value]="number(r.totals.views)" [delta]="change(r.totals.views, r.previous.views)" [hint]="'cabinet.dashboard.vsPrevious' | transloco" />
          <app-stat-tile [label]="'cabinet.stats.contacts' | transloco" icon="icon-[heroicons--chat-bubble-left-ellipsis]" [value]="number(r.totals.contacts)" [delta]="change(r.totals.contacts, r.previous.contacts)" [hint]="'cabinet.dashboard.vsPrevious' | transloco" />
          <app-stat-tile class="col-span-2 lg:col-span-1" [label]="'cabinet.stats.conversion' | transloco" icon="icon-[heroicons--funnel]" [value]="share(r.totals.contacts, r.totals.views)" [hint]="'cabinet.stats.conversionExplained' | transloco" />
        </div>

        <div class="ui-card grid grid-cols-1 gap-6 p-4 sm:grid-cols-2">
          <app-daily-bar-chart [title]="'cabinet.stats.viewsPerDay' | transloco" [points]="viewsPoints()" />
          <app-daily-bar-chart [title]="'cabinet.stats.contactsPerDay' | transloco" [points]="contactsPoints()" />
          <p class="text-xs text-text-muted sm:col-span-2">{{ 'cabinet.stats.howCounted' | transloco }}</p>
        </div>

        <section class="flex flex-col gap-3" aria-labelledby="channels">
          <h2 id="channels" class="font-semibold text-text">{{ 'cabinet.stats.channels' | transloco }}</h2>
          <ul class="ui-card flex flex-col gap-3 p-4">
            @for (channel of channels(); track channel.key) {
              <li class="flex items-center gap-3 text-sm">
                <span class="flex w-28 shrink-0 items-center gap-2 text-text">
                  <span [class]="channel.icon + ' size-4 text-text-muted'" aria-hidden="true"></span>{{ channel.label }}
                </span>
                <span class="h-2 flex-1 overflow-hidden rounded-full bg-text/10" aria-hidden="true">
                  <span class="block h-full rounded-full bg-brand-600" [style.width.%]="(channel.value / maxChannel()) * 100"></span>
                </span>
                <span class="w-10 shrink-0 text-right text-text tabular-nums">{{ channel.value }}</span>
              </li>
            }
          </ul>
        </section>

        <section class="flex flex-col gap-3" aria-labelledby="by-listing">
          <h2 id="by-listing" class="font-semibold text-text">{{ 'cabinet.stats.byListing' | transloco }}</h2>
          @if (rows().length > 0) {
            <div class="ui-table-wrap">
              <table class="ui-table min-w-[520px]">
                <thead>
                  <tr>
                    <th scope="col">{{ 'cabinet.stats.listing' | transloco }}</th>
                    <th scope="col" class="text-right">{{ 'cabinet.stats.views' | transloco }}</th>
                    <th scope="col" class="text-right">{{ 'cabinet.stats.contacts' | transloco }}</th>
                    <th scope="col" class="text-right">{{ 'cabinet.stats.conversion' | transloco }}</th>
                  </tr>
                </thead>
                <tbody>
                  @for (item of rows(); track item.listing.id) {
                    <tr>
                      <td class="max-w-0">
                        <a [appSiteLink]="path(item.listing)" [lang]="lang(item.listing.locale)" class="block truncate font-medium text-text hover:underline">{{ item.listing.title }}</a>
                        <span class="text-xs text-text-muted">{{ 'cabinet.kinds.' + item.listing.targetType | transloco }}</span>
                      </td>
                      <td class="text-right text-text tabular-nums">{{ item.views }}</td>
                      <td class="text-right text-text tabular-nums">{{ item.contacts }}</td>
                      <td class="text-right text-text-muted tabular-nums">{{ share(item.contacts, item.views) }}</td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          } @else {
            <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-10 text-center">
              <p class="text-sm text-text-muted">{{ 'cabinet.stats.noListings' | transloco }}</p>
              <a appSiteLink="/partner/listings" class="ui-btn-primary">{{ 'cabinet.nav.listings' | transloco }}</a>
            </div>
          }
        </section>
      }
    </div>
  `,
})
export class PartnerStatsPage {
  private readonly transloco = inject(TranslocoService);
  private readonly document = inject(DOCUMENT);
  private readonly currentLang = injectLang();
  protected readonly periods = PERIODS;
  protected readonly days = signal<number>(30);

  private readonly stats = httpResource<StatsReport>(() => ({
    url: '/listing-stats/mine',
    params: { days: this.days() },
  }));
  // The last loaded report stays while another period loads.
  protected readonly report = linkedSignal<StatsReport | undefined, StatsReport | undefined>({
    source: () => loaded(this.stats),
    computation: (next, previous) => next ?? previous?.value,
  });
  protected readonly viewsPoints = computed(
    () => this.report()?.daily.map((d) => ({ day: d.day, value: d.views })) ?? [],
  );
  protected readonly contactsPoints = computed(
    () => this.report()?.daily.map((d) => ({ day: d.day, value: d.contacts })) ?? [],
  );
  protected readonly rows = computed(() =>
    [...(this.report()?.listings ?? [])].sort(
      (a, b) => b.views - a.views || b.contacts - a.contacts,
    ),
  );
  protected readonly channels = computed(() => {
    const totals = this.report()?.totals;
    if (!totals) return [];
    const t = (key: string): string => this.transloco.translate(key);
    return [
      { key: 'whatsapp', label: 'WhatsApp', icon: 'icon-[mdi--whatsapp]', value: totals.whatsapp },
      { key: 'telegram', label: 'Telegram', icon: 'icon-[mdi--telegram]', value: totals.telegram },
      {
        key: 'phone',
        label: t('cabinet.stats.phone'),
        icon: 'icon-[heroicons--phone]',
        value: totals.phone,
      },
      {
        key: 'website',
        label: t('cabinet.stats.website'),
        icon: 'icon-[heroicons--globe-alt]',
        value: totals.website,
      },
    ];
  });
  protected readonly maxChannel = computed(() =>
    Math.max(1, ...this.channels().map((channel) => channel.value)),
  );

  protected readonly change = percentChange;
  protected readonly share = shareOf;
  protected readonly path = listingPath;
  protected readonly lang = contentLang;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.stats'),
      description: this.transloco.translate('cabinet.stats.subtitle'),
      robots: 'noindex',
    }));
  }

  protected number(value: number): string {
    return formatAmount(this.currentLang(), value);
  }

  protected exportCsv(): void {
    const report = this.report();
    if (!report) return;
    const t = (key: string): string => this.transloco.translate(key);
    downloadCsv(this.document, `ermekten-stats-${report.from}-${report.to}.csv`, [
      [
        t('cabinet.stats.listing'),
        t('cabinet.stats.views'),
        'WhatsApp',
        'Telegram',
        t('cabinet.stats.phone'),
        t('cabinet.stats.website'),
        t('cabinet.stats.contacts'),
      ],
      ...this.rows().map((row) => [
        row.listing.title,
        row.views,
        row.whatsapp,
        row.telegram,
        row.phone,
        row.website,
        row.contacts,
      ]),
    ]);
  }
}
