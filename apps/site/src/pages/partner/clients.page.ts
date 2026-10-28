import type { CrmClient, Paginated } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { ClientForm } from '@/pages/partner/client-form';
import { clientName } from '@/pages/partner/crm';
import { ListSkeleton } from '@/ui/list-skeleton';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const LIMIT = 50;

// /partner/clients: everyone who signed up for the partner's tours and the
// people they added by hand — found by name, phone, Telegram or note and
// by tag, with how many tours they went on and what they still owe.
@Component({
  selector: 'app-partner-clients-page',
  imports: [ClientForm, ListSkeleton, PersonAvatar, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.clients' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'crm.clients.subtitle' | transloco }}</p>
        </div>
        @if (!adding()) {
          <button type="button" class="ui-btn-primary" (click)="adding.set(true)"><span class="icon-[heroicons--user-plus] size-4" aria-hidden="true"></span>{{ 'crm.client.add' | transloco }}</button>
        }
      </div>

      @if (adding()) {
        <section class="ui-card p-4" aria-labelledby="new-client">
          <h2 id="new-client" class="mb-3 font-semibold text-text">{{ 'crm.client.new' | transloco }}</h2>
          <app-client-form (saved)="created($event)" (cancelled)="adding.set(false)" />
        </section>
      }

      <div class="flex flex-col gap-2">
        <div class="relative">
          <span class="icon-[heroicons--magnifying-glass] pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
          <input type="search" class="ui-input pl-9" [value]="query()" (input)="search($any($event.target).value)" [placeholder]="'crm.clients.search' | transloco" [attr.aria-label]="'crm.clients.search' | transloco" />
        </div>
        @if (tags().length > 0) {
          <div class="flex flex-wrap gap-1.5" role="group" [attr.aria-label]="'crm.client.tags' | transloco">
            @for (t of tags(); track t) {
              <button type="button" class="ui-pill" [attr.aria-pressed]="tag() === t" (click)="tag.set(tag() === t ? null : t)">#{{ t }}</button>
            }
          </div>
        }
      </div>

      @if (clients.isLoading() && !clients.hasValue()) {
        <app-list-skeleton variant="rows" [count]="5" />
      } @else if (list().length > 0) {
        <p class="text-sm text-text-muted">{{ 'crm.clients.count' | transloco: { count: total() } }}</p>
        <ul class="ui-card flex flex-col divide-y divide-border">
          @for (client of list(); track client.id) {
            <li>
              <a [appSiteLink]="'/partner/clients/' + client.id" class="flex items-center gap-3 px-4 py-3 hover:bg-text/3">
                @if (client.account) {
                  <app-person-avatar [person]="client.account" [size]="40" />
                } @else {
                  <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-text/7 text-text-muted"><span class="icon-[heroicons--user] size-5" aria-hidden="true"></span></span>
                }
                <span class="min-w-0 flex-1">
                  <span class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <span class="truncate font-medium text-text">{{ name(client) }}</span>
                    @if (!client.account) {
                      <span class="ui-badge ui-badge-muted">{{ 'crm.client.manual' | transloco }}</span>
                    }
                    @for (t of client.tags; track t) {
                      <span class="text-xs text-brand-700 dark:text-brand-300">#{{ t }}</span>
                    }
                  </span>
                  <span class="block truncate text-xs text-text-muted">
                    {{ contacts(client) }}
                  </span>
                </span>
                <span class="hidden shrink-0 text-right text-xs text-text-muted sm:block">
                  <span class="block">{{ 'crm.clients.tours' | transloco: { count: client.stats.tours } }}</span>
                  @if (client.stats.lastTour) {
                    <span class="block">{{ date(client.stats.lastTour) }}</span>
                  }
                </span>
                @if (client.stats.due > client.stats.paid) {
                  <span class="ui-badge ui-badge-warning shrink-0">{{ 'crm.owes' | transloco: { amount: amount(client.stats.due - client.stats.paid) } }}</span>
                }
              </a>
            </li>
          }
        </ul>
        @if (total() > list().length) {
          <button type="button" class="ui-btn-soft self-center" (click)="limit.set(limit() + LIMIT)">{{ 'crm.clients.more' | transloco }}</button>
        }
      } @else if (clients.hasValue()) {
        <app-state-block icon="icon-[heroicons--users]" [title]="query() || tag() ? 'crm.clients.notFound' : 'crm.clients.empty'" text="crm.clients.emptyHint" />
      }
    </div>
  `,
})
export class PartnerClientsPage {
  private readonly transloco = inject(TranslocoService);
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();
  protected readonly LIMIT = LIMIT;

  protected readonly adding = signal(false);
  protected readonly query = signal('');
  private readonly debounced = signal('');
  protected readonly tag = signal<string | null>(null);
  protected readonly limit = signal(LIMIT);
  private timer: ReturnType<typeof setTimeout> | undefined;

  protected readonly clients = httpResource<Paginated<CrmClient>>(() => ({
    url: '/crm/clients',
    params: {
      limit: this.limit(),
      ...(this.debounced() ? { q: this.debounced() } : {}),
      ...(this.tag() ? { tag: this.tag() as string } : {}),
    },
  }));
  private readonly tagList = httpResource<string[]>(() => '/crm/clients/tags');
  protected readonly tags = computed(() => loaded(this.tagList) ?? []);
  protected readonly list = computed(() => loaded(this.clients)?.items ?? []);
  protected readonly total = computed(() => loaded(this.clients)?.total ?? 0);

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.clients'),
      description: this.transloco.translate('crm.clients.subtitle'),
      robots: 'noindex',
    }));
  }

  protected search(value: string): void {
    this.query.set(value);
    clearTimeout(this.timer);
    this.timer = setTimeout(() => {
      this.debounced.set(value.trim());
      this.limit.set(LIMIT);
    }, 250);
  }

  protected name(client: CrmClient): string {
    return clientName(client);
  }

  protected contacts(client: CrmClient): string {
    const parts = [client.phone, client.telegram ? `@${client.telegram}` : null];
    if (!client.telegram && client.account?.username) parts.push(`@${client.account.username}`);
    return parts.filter(Boolean).join(' · ') || this.transloco.translate('crm.client.noContacts');
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  protected amount(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected async created(client: CrmClient): Promise<void> {
    this.adding.set(false);
    this.tagList.reload();
    await this.navigate(`/partner/clients/${client.id}`);
  }
}
