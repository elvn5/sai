import type { CrmBroadcast, CrmTemplate, Tour } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { BroadcastForm } from '@/pages/partner/broadcast-form';
import { injectConfirm } from '@/ui/confirm-dialog';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/broadcasts: a message to a tour's group, clients with a tag or
// all clients (into each one's chat), the ones sent before, and the
// templates — texts sent again and again (also offered in chats).
@Component({
  selector: 'app-partner-broadcasts-page',
  imports: [BroadcastForm, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div>
        <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.broadcasts' | transloco }}</h1>
        <p class="text-sm text-text-muted">{{ 'crm.broadcast.subtitle' | transloco }}</p>
      </div>

      <section class="ui-card p-4" aria-labelledby="new-broadcast">
        <h2 id="new-broadcast" class="mb-3 font-semibold text-text">{{ 'crm.broadcast.new' | transloco }}</h2>
        <app-broadcast-form [tags]="tags()" [tours]="upcoming()" (sent)="history.reload()" />
      </section>

      <section class="flex flex-col gap-2" aria-labelledby="templates">
        <h2 id="templates" class="font-semibold text-text">{{ 'crm.templates.title' | transloco }}</h2>
        <ul class="ui-card flex flex-col divide-y divide-border">
          @for (t of templateList(); track t.id) {
            <li class="flex items-start gap-3 px-4 py-3 text-sm">
              <div class="min-w-0 flex-1">
                <p class="font-medium text-text">{{ t.title }}</p>
                <p class="line-clamp-3 whitespace-pre-line text-text-muted">{{ t.body }}</p>
              </div>
              <button type="button" class="ui-icon-btn size-8" [attr.aria-label]="'listing.edit' | transloco" (click)="edit(t)"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span></button>
              <button type="button" class="ui-icon-btn size-8" [attr.aria-label]="'crm.templates.delete' | transloco" (click)="remove(t)"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span></button>
            </li>
          } @empty {
            <li class="px-4 py-3 text-sm text-text-muted">{{ 'crm.templates.empty' | transloco }}</li>
          }
          <li class="px-4 py-3">
            <form class="flex flex-col gap-2" (submit)="save($event, title.value, body.value)">
              <label class="sr-only" for="template-title">{{ 'crm.templates.name' | transloco }}</label>
              <input #title id="template-title" class="ui-input" maxlength="100" [value]="editing()?.title ?? ''" [placeholder]="'crm.templates.name' | transloco" />
              <label class="sr-only" for="template-body">{{ 'crm.broadcast.text' | transloco }}</label>
              <textarea #body id="template-body" class="ui-input" rows="3" maxlength="4000" [value]="editing()?.body ?? ''" [placeholder]="'crm.templates.bodyPlaceholder' | transloco"></textarea>
              <div class="flex gap-2">
                <button type="submit" class="ui-btn-soft" [disabled]="busy()">{{ (editing() ? 'crm.save' : 'crm.templates.add') | transloco }}</button>
                @if (editing()) {
                  <button type="button" class="ui-btn-ghost" (click)="editing.set(null)">{{ 'crm.cancel' | transloco }}</button>
                }
              </div>
            </form>
          </li>
        </ul>
      </section>

      @if (sent().length > 0) {
        <section class="flex flex-col gap-2" aria-labelledby="sent-broadcasts">
          <h2 id="sent-broadcasts" class="font-semibold text-text">{{ 'crm.broadcast.history' | transloco }}</h2>
          <ul class="ui-card flex flex-col divide-y divide-border">
            @for (b of sent(); track b.id) {
              <li class="flex flex-col gap-1 px-4 py-3 text-sm">
                <p class="flex flex-wrap gap-x-2 text-xs text-text-muted">
                  <span>{{ when(b.createdAt) }}</span>
                  <span>{{ audience(b.audience) }}</span>
                  <span>{{ 'crm.broadcast.result' | transloco: { sent: b.sent, skipped: b.skipped } }}</span>
                </p>
                <p class="line-clamp-3 whitespace-pre-line text-text">{{ b.body }}</p>
              </li>
            }
          </ul>
        </section>
      }
    </div>
  `,
})
export class PartnerBroadcastsPage {
  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly busy = signal(false);
  protected readonly editing = signal<CrmTemplate | null>(null);

  protected readonly history = httpResource<CrmBroadcast[]>(() => '/crm/broadcasts');
  protected readonly sent = computed(() => loaded(this.history) ?? []);
  private readonly templates = httpResource<CrmTemplate[]>(() => '/crm/templates');
  protected readonly templateList = computed(() => loaded(this.templates) ?? []);
  private readonly tagList = httpResource<string[]>(() => '/crm/clients/tags');
  protected readonly tags = computed(() => loaded(this.tagList) ?? []);
  private readonly tourList = httpResource<Tour[]>(() => '/tours/mine');
  protected readonly upcoming = computed(() => {
    const today = todayInBishkek();
    return (loaded(this.tourList) ?? [])
      .filter((t) => t.endDate >= today)
      .map((t) => ({ id: t.id, title: `${t.title} · ${t.startDate}` }));
  });

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.broadcasts'),
      description: this.transloco.translate('crm.broadcast.subtitle'),
      robots: 'noindex',
    }));
  }

  protected when(iso: string): string {
    return formatInBishkek(this.lang(), iso, {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  protected audience(value: string): string {
    const [kind, rest] = value.split(':');
    if (kind === 'tag') return this.transloco.translate('crm.broadcast.tag', { tag: rest });
    if (kind === 'tour') {
      const tour = (loaded(this.tourList) ?? []).find((t) => t.id === rest);
      return this.transloco.translate('crm.broadcast.tour', { title: tour?.title ?? '—' });
    }
    return this.transloco.translate('crm.broadcast.all');
  }

  protected edit(t: CrmTemplate): void {
    this.editing.set(t);
  }

  protected async save(event: Event, title: string, body: string): Promise<void> {
    event.preventDefault();
    if (!title.trim() || !body.trim()) {
      this.notify.warning({ title: 'crm.templates.required' });
      return;
    }
    const current = this.editing();
    this.busy.set(true);
    try {
      const list = await firstValueFrom(
        current
          ? this.http.put<CrmTemplate[]>(`/crm/templates/${current.id}`, { title, body })
          : this.http.post<CrmTemplate[]>('/crm/templates', { title, body }),
      );
      this.templates.set(list);
      this.editing.set(null);
      (event.target as HTMLFormElement).reset();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(t: CrmTemplate): Promise<void> {
    if (!(await this.confirm({ title: 'crm.templates.deleteConfirm', danger: true }))) return;
    try {
      await firstValueFrom(this.http.delete(`/crm/templates/${t.id}`));
      this.templates.reload();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    }
  }
}
