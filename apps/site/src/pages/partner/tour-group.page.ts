import type { CrmClient, CrmGroup, Paginated, ParticipantStatus } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange } from '@/core/lib/format';
import { contentLang, downloadCsv } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { ClientForm } from '@/pages/partner/client-form';
import { clientName, groupRows } from '@/pages/partner/crm';
import { ParticipantRow } from '@/pages/partner/participant-row';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { DOCUMENT } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/tours/:id/participants: a tour's group — who has a place, the
// waitlist and the cancelled, check-in, what each owes and has paid, and
// adding clients (from the base or new ones). Exports the group as CSV.
@Component({
  selector: 'app-partner-tour-group-page',
  imports: [ClientForm, ParticipantRow, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <a appSiteLink="/partner/listings" class="flex items-center gap-1 self-start text-sm text-text-muted hover:text-text">
        <span class="icon-[heroicons--arrow-left] size-4" aria-hidden="true"></span>{{ 'cabinet.nav.listings' | transloco }}
      </a>
      @if (data(); as g) {
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="min-w-0">
            <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ g.tour.title }}</h1>
            <p class="text-sm text-text-muted">
              {{ dates() }}
              · <a [appSiteLink]="'/tours/' + g.tour.slug" [lang]="tourLang()" class="hover:underline">{{ 'crm.group.tourPage' | transloco }}</a>
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="ui-btn-soft" [disabled]="g.participants.length === 0" (click)="exportCsv(g)"><span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>{{ 'crm.group.export' | transloco }}</button>
            @if (!adding()) {
              <button type="button" class="ui-btn-primary" (click)="adding.set(true)"><span class="icon-[heroicons--user-plus] size-4" aria-hidden="true"></span>{{ 'crm.group.add' | transloco }}</button>
            }
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-5">
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.group.places' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ g.totals.active }}@if (g.tour.capacity !== null) {<span class="text-base font-semibold text-text-muted"> / {{ g.tour.capacity }}</span>}</p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.status.waitlist' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ g.totals.waitlist }}</p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.paid' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ amount(g.totals.paid) }}<span class="text-base font-semibold text-text-muted"> / {{ amount(g.totals.due) }}</span></p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.group.forms' | transloco }}</p><p class="text-xl font-extrabold tabular-nums" [class]="g.totals.forms < g.totals.active ? 'text-amber-700 dark:text-amber-300' : 'text-text'">{{ g.totals.forms }} / {{ g.totals.active }}</p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.group.arrived' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ g.totals.arrived }} / {{ g.totals.active }}</p></div>
        </div>

        @if (adding()) {
          <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="add-participant">
            <div class="flex items-center justify-between gap-2">
              <h2 id="add-participant" class="font-semibold text-text">{{ 'crm.group.add' | transloco }}</h2>
              <button type="button" class="ui-icon-btn" [attr.aria-label]="'crm.cancel' | transloco" (click)="closeAdd()"><span class="icon-[heroicons--x-mark] size-5" aria-hidden="true"></span></button>
            </div>
            @if (full()) {
              <p class="ui-banner ui-banner-warning text-sm">{{ 'crm.group.fullHint' | transloco }}</p>
            }
            @if (newClient()) {
              <app-client-form (saved)="createdClient($event)" (cancelled)="newClient.set(false)" />
            } @else {
              <input type="search" class="ui-input" [value]="pick()" (input)="pick.set($any($event.target).value)" [placeholder]="'crm.clients.search' | transloco" [attr.aria-label]="'crm.clients.search' | transloco" />
              <ul class="flex max-h-72 flex-col divide-y divide-border overflow-y-auto rounded-lg border border-border">
                @for (c of candidates(); track c.id) {
                  <li class="flex items-center gap-3 px-3 py-2">
                    <span class="min-w-0 flex-1 truncate text-sm text-text">{{ name(c) }}</span>
                    <button type="button" class="ui-btn-soft ui-btn-xs" [disabled]="busy()" (click)="add(c.id)">{{ (full() ? 'crm.group.addWaitlist' : 'crm.group.addOne') | transloco }}</button>
                  </li>
                } @empty {
                  <li class="px-3 py-3 text-sm text-text-muted">{{ 'crm.clients.notFound' | transloco }}</li>
                }
              </ul>
              <button type="button" class="ui-btn-ghost self-start" (click)="newClient.set(true)"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'crm.group.newClient' | transloco }}</button>
            }
          </section>
        }

        @for (section of sections(); track section.key) {
          @if (section.items.length > 0) {
            <section class="flex flex-col gap-2" [attr.aria-label]="section.key | transloco">
              <h2 class="font-semibold text-text">{{ section.key | transloco }} <span class="text-text-muted">({{ section.items.length }})</span></h2>
              <ul class="ui-card flex flex-col divide-y divide-border">
                @for (p of section.items; track p.id; let i = $index) {
                  <li><app-participant-row [tourId]="g.tour.id" [participant]="p" [index]="i + 1" (changed)="set($event)" /></li>
                }
              </ul>
            </section>
          }
        }
        @if (g.participants.length === 0 && !adding()) {
          <app-state-block icon="icon-[heroicons--user-group]" title="crm.group.empty" text="crm.group.emptyHint">
            <button type="button" class="ui-btn-primary" (click)="adding.set(true)">{{ 'crm.group.add' | transloco }}</button>
          </app-state-block>
        }
      } @else if (group.error()) {
        <app-state-block icon="icon-[heroicons--user-group]" tone="danger" title="crm.group.notFound" />
      }
    </div>
  `,
})
export class PartnerTourGroupPage {
  readonly id = input.required<string>();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly document = inject(DOCUMENT);
  private readonly lang = injectLang();

  protected readonly group = httpResource<CrmGroup>(() => `/crm/tours/${this.id()}/participants`);
  protected readonly data = computed(() => loaded(this.group) ?? null);
  protected readonly adding = signal(false);
  protected readonly newClient = signal(false);
  protected readonly pick = signal('');
  protected readonly busy = signal(false);

  // The partner's clients to add (once the panel opens).
  private readonly clients = httpResource<Paginated<CrmClient>>(() =>
    this.adding() ? { url: '/crm/clients', params: { limit: 200 } } : undefined,
  );
  protected readonly candidates = computed(() => {
    const on = new Set(
      (this.data()?.participants ?? [])
        .filter((p) => p.status !== 'cancelled')
        .map((p) => p.clientId),
    );
    const query = this.pick().trim().toLowerCase();
    return (loaded(this.clients)?.items ?? [])
      .filter((c) => !on.has(c.id))
      .filter(
        (c) =>
          !query ||
          `${clientName(c)} ${c.phone ?? ''} ${c.telegram ?? ''}`.toLowerCase().includes(query),
      )
      .slice(0, 50);
  });

  protected readonly full = computed(() => {
    const g = this.data();
    return !!g && g.tour.capacity !== null && g.totals.active >= g.tour.capacity;
  });
  protected readonly sections = computed(() => {
    const all = this.data()?.participants ?? [];
    return [
      {
        key: 'crm.group.withPlace',
        items: all.filter((p) => p.status === 'new' || p.status === 'confirmed'),
      },
      { key: 'crm.status.waitlist', items: all.filter((p) => p.status === 'waitlist') },
      { key: 'crm.status.cancelled', items: all.filter((p) => p.status === 'cancelled') },
    ];
  });
  protected readonly dates = computed(() => {
    const t = this.data()?.tour;
    return t ? formatDateRange(this.lang(), t.startDate, t.endDate) : '';
  });
  protected readonly tourLang = computed(() => contentLang(this.data()?.tour.locale ?? 'ru'));

  constructor() {
    bindPageMeta(() => ({
      title: this.data()?.tour.title ?? this.transloco.translate('crm.group.title'),
      description: this.transloco.translate('crm.group.title'),
      robots: 'noindex',
    }));
  }

  protected name(c: CrmClient): string {
    return clientName(c);
  }

  protected amount(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected set(group: CrmGroup): void {
    this.group.set(group);
  }

  protected closeAdd(): void {
    this.adding.set(false);
    this.newClient.set(false);
    this.pick.set('');
  }

  protected async add(clientId: string): Promise<void> {
    this.busy.set(true);
    try {
      const status: ParticipantStatus = this.full() ? 'waitlist' : 'confirmed';
      this.set(
        await firstValueFrom(
          this.http.post<CrmGroup>(`/crm/tours/${this.id()}/participants`, { clientId, status }),
        ),
      );
      this.notify.success({
        title: status === 'waitlist' ? 'crm.group.addedWaitlist' : 'crm.group.added',
      });
    } catch (error) {
      this.notify.error({
        title:
          error instanceof HttpErrorResponse && error.status === 409
            ? 'crm.errors.conflict'
            : 'crm.saveError',
      });
    } finally {
      this.busy.set(false);
    }
  }

  protected async createdClient(client: CrmClient): Promise<void> {
    this.newClient.set(false);
    this.clients.reload();
    await this.add(client.id);
  }

  protected exportCsv(group: CrmGroup): void {
    const t = (key: string): string => this.transloco.translate(key);
    const rows = groupRows(group, {
      headers: [
        'number',
        'name',
        'phone',
        'telegram',
        'status',
        'arrived',
        'due',
        'paid',
        'debt',
        'emergency',
        'experience',
        'health',
        'waiver',
      ].map((h) => t(`crm.csv.${h}`)),
      status: (s) => t(`crm.status.${s}`),
      experience: (e) => t(`participantForm.levels.${e}.title`),
      yes: t('crm.csv.yes'),
    });
    downloadCsv(this.document, `${group.tour.slug}-${group.tour.startDate}.csv`, rows);
  }
}
