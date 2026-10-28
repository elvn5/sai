import type { CrmClient } from '@/core/api/types';
import { openChat } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { ClientForm } from '@/pages/partner/client-form';
import { clientName, statusBadge } from '@/pages/partner/crm';
import { injectConfirm } from '@/ui/confirm-dialog';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/clients/:id: one client — contacts, note and tags (editable),
// their tours with the partner with what they owe, and the chat with them
// (accounts only).
@Component({
  selector: 'app-partner-client-page',
  imports: [ClientForm, PersonAvatar, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <a appSiteLink="/partner/clients" class="flex items-center gap-1 self-start text-sm text-text-muted hover:text-text">
        <span class="icon-[heroicons--arrow-left] size-4" aria-hidden="true"></span>{{ 'cabinet.nav.clients' | transloco }}
      </a>
      @if (data(); as c) {
        <section class="ui-card flex flex-col gap-4 p-4">
          <div class="flex flex-wrap items-center gap-3">
            @if (c.account) {
              <app-person-avatar [person]="c.account" [size]="56" />
            } @else {
              <span class="flex size-14 items-center justify-center rounded-full bg-text/7 text-text-muted"><span class="icon-[heroicons--user] size-7" aria-hidden="true"></span></span>
            }
            <div class="min-w-0 flex-1">
              <h1 class="truncate text-xl font-extrabold text-text">{{ name(c) }}</h1>
              <p class="text-sm text-text-muted">
                {{ (c.account ? 'crm.client.account' : 'crm.client.manualLong') | transloco }}
                · {{ 'crm.client.since' | transloco: { date: date(c.createdAt.slice(0, 10)) } }}
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              @if (c.userId) {
                <button type="button" class="ui-btn-primary" [disabled]="busy()" (click)="write(c)"><span class="icon-[heroicons--chat-bubble-left-right] size-4" aria-hidden="true"></span>{{ 'chat.write' | transloco }}</button>
              }
              @if (!editing()) {
                <button type="button" class="ui-btn-soft" (click)="editing.set(true)"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</button>
              }
              @if (!c.userId) {
                <button type="button" class="ui-btn-ghost" [disabled]="busy()" [attr.aria-label]="'crm.client.delete' | transloco" (click)="remove(c)"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span></button>
              }
            </div>
          </div>

          @if (editing()) {
            <app-client-form [client]="c" (saved)="saved()" (cancelled)="editing.set(false)" />
          } @else {
            <dl class="grid gap-3 text-sm sm:grid-cols-3">
              <div><dt class="text-text-muted">{{ 'crm.client.phone' | transloco }}</dt><dd class="text-text">@if (c.phone) {<a class="hover:underline" [href]="'tel:' + c.phone">{{ c.phone }}</a>} @else {—}</dd></div>
              <div><dt class="text-text-muted">{{ 'crm.client.telegram' | transloco }}</dt><dd class="text-text">@if (telegram(c); as tg) {<a class="hover:underline" [href]="'https://t.me/' + tg" target="_blank" rel="noopener">&#64;{{ tg }}</a>} @else {—}</dd></div>
              <div><dt class="text-text-muted">{{ 'crm.client.tags' | transloco }}</dt><dd class="text-text">@for (t of c.tags; track t) {<span class="mr-1.5 text-brand-700 dark:text-brand-300">#{{ t }}</span>} @empty {—}</dd></div>
            </dl>
            @if (c.note) {
              <p class="rounded-lg bg-text/4 px-3 py-2 text-sm whitespace-pre-line text-text">{{ c.note }}</p>
            }
          }
        </section>

        <div class="grid grid-cols-3 gap-3">
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.client.toursCount' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ c.stats.tours }}</p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.paid' | transloco }}</p><p class="text-xl font-extrabold text-text tabular-nums">{{ amount(c.stats.paid) }}</p></div>
          <div class="ui-card px-4 py-3"><p class="text-xs text-text-muted">{{ 'crm.debt' | transloco }}</p><p class="text-xl font-extrabold tabular-nums" [class]="c.stats.due > c.stats.paid ? 'text-amber-700 dark:text-amber-300' : 'text-text'">{{ amount(max0(c.stats.due - c.stats.paid)) }}</p></div>
        </div>

        <section class="flex flex-col gap-2" aria-labelledby="client-tours">
          <h2 id="client-tours" class="font-semibold text-text">{{ 'crm.client.history' | transloco }}</h2>
          @if (c.history?.length) {
            <ul class="ui-card flex flex-col divide-y divide-border">
              @for (h of c.history; track h.participantId) {
                <li>
                  <a [appSiteLink]="'/partner/tours/' + h.tourId + '/participants'" class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 hover:bg-text/3">
                    <span class="min-w-0 flex-1">
                      <span class="block truncate font-medium text-text">{{ h.title }}</span>
                      <span class="text-xs text-text-muted">{{ date(h.startDate) }}</span>
                    </span>
                    <span [class]="statusBadge(h.status)">{{ 'crm.status.' + h.status | transloco }}</span>
                    @if (h.arrived) {
                      <span class="ui-badge ui-badge-success"><span class="icon-[heroicons--check]" aria-hidden="true"></span>{{ 'tours.signup.arrived' | transloco }}</span>
                    }
                    <span class="text-sm text-text-muted tabular-nums">{{ amount(h.paid) }} / {{ amount(h.due) }}</span>
                  </a>
                </li>
              }
            </ul>
          } @else {
            <p class="text-sm text-text-muted">{{ 'crm.client.noTours' | transloco }}</p>
          }
        </section>
      } @else if (client.error()) {
        <app-state-block icon="icon-[heroicons--user]" tone="danger" title="crm.client.notFound" />
      }
    </div>
  `,
})
export class PartnerClientPage {
  readonly id = input.required<string>();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  protected readonly client = httpResource<CrmClient>(() => `/crm/clients/${this.id()}`);
  protected readonly data = computed(() => loaded(this.client) ?? null);
  protected readonly editing = signal(false);
  protected readonly busy = signal(false);
  protected readonly statusBadge = statusBadge;

  constructor() {
    bindPageMeta(() => ({
      title: this.data()
        ? clientName(this.data() as CrmClient)
        : this.transloco.translate('cabinet.nav.clients'),
      description: this.transloco.translate('crm.clients.subtitle'),
      robots: 'noindex',
    }));
  }

  protected name(c: CrmClient): string {
    return clientName(c);
  }

  protected telegram(c: CrmClient): string | null {
    return c.telegram ?? c.account?.username ?? null;
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  protected amount(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected max0(value: number): number {
    return Math.max(value, 0);
  }

  protected saved(): void {
    this.editing.set(false);
    this.client.reload();
    this.notify.success({ title: 'crm.saved' });
  }

  protected async write(c: CrmClient): Promise<void> {
    if (!c.userId) return;
    this.busy.set(true);
    try {
      const chat = c.chatId ? { id: c.chatId } : await openChat(this.http, { userId: c.userId });
      await this.navigate(`/partner/messages/${chat.id}`);
    } catch {
      this.notify.error({ title: 'chat.openError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(c: CrmClient): Promise<void> {
    if (
      !(await this.confirm({
        title: 'crm.client.deleteConfirm',
        text: 'crm.client.deleteHint',
        params: { name: clientName(c) },
        danger: true,
      }))
    )
      return;
    this.busy.set(true);
    try {
      await firstValueFrom(this.http.delete(`/crm/clients/${c.id}`));
      this.notify.success({ title: 'crm.client.deleted' });
      await this.navigate('/partner/clients');
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }
}
