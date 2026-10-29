import type { CrmClient, CrmLead, Paginated, Tour } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { LEAD_STAGES, clientName } from '@/pages/partner/crm';
import { LeadCard } from '@/pages/partner/lead-card';
import { StateBlock } from '@/ui/state-block';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/leads: inquiries on their way to a booking, a column per stage
// (new → working → offer → won / lost). A traveler writing to the partner
// opens one by itself; signing up for the partner's tour wins it.
@Component({
  selector: 'app-partner-leads-page',
  imports: [LeadCard, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.leads' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'crm.leads.subtitle' | transloco }}</p>
        </div>
        @if (!adding()) {
          <button type="button" class="ui-btn-primary" (click)="adding.set(true)"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'crm.leads.add' | transloco }}</button>
        }
      </div>

      @if (adding()) {
        <form class="ui-card grid gap-3 p-4 sm:grid-cols-[1fr_1fr_1.5fr_auto] sm:items-end" (submit)="create($event, client.value, tour.value, note.value)">
          <div class="ui-field">
            <label class="ui-label" for="lead-client">{{ 'crm.leads.client' | transloco }}</label>
            <select #client id="lead-client" class="ui-input">
              @for (c of clients(); track c.id) {
                <option [value]="c.id">{{ name(c) }}</option>
              }
            </select>
          </div>
          <div class="ui-field">
            <label class="ui-label" for="lead-tour">{{ 'crm.leads.tour' | transloco }}</label>
            <select #tour id="lead-tour" class="ui-input">
              <option value="">—</option>
              @for (t of tours(); track t.id) {
                <option [value]="t.id">{{ t.title }} · {{ t.startDate }}</option>
              }
            </select>
          </div>
          <div class="ui-field">
            <label class="ui-label" for="lead-note">{{ 'crm.payment.note' | transloco }}</label>
            <input #note id="lead-note" class="ui-input" maxlength="2000" [placeholder]="'crm.leads.notePlaceholder' | transloco" />
          </div>
          <div class="flex gap-2">
            <button type="submit" class="ui-btn-primary" [disabled]="busy() || clients().length === 0">{{ 'crm.leads.add' | transloco }}</button>
            <button type="button" class="ui-btn-ghost" (click)="adding.set(false)">{{ 'crm.cancel' | transloco }}</button>
          </div>
          @if (clients().length === 0) {
            <p class="ui-hint sm:col-span-4">{{ 'crm.leads.noClients' | transloco }}</p>
          }
        </form>
      }

      @if (leads.hasValue() && all().length === 0) {
        <app-state-block icon="icon-[heroicons--funnel]" title="crm.leads.empty" text="crm.leads.emptyHint" />
      } @else {
        <div class="-mx-4 flex gap-3 overflow-x-auto px-4 pb-2 lg:mx-0 lg:grid lg:grid-cols-5 lg:overflow-visible lg:px-0">
          @for (column of columns(); track column.stage) {
            <section class="flex w-64 shrink-0 flex-col gap-2 lg:w-auto" [attr.aria-label]="'crm.leads.stages.' + column.stage | transloco">
              <h2 class="flex items-center justify-between text-sm font-semibold text-text">
                {{ 'crm.leads.stages.' + column.stage | transloco }}
                <span class="text-text-muted">{{ column.items.length }}</span>
              </h2>
              @for (lead of column.items; track lead.id) {
                <app-lead-card [lead]="lead" (changed)="changed(lead.id, $event)" />
              }
            </section>
          }
        </div>
      }
    </div>
  `,
})
export class PartnerLeadsPage {
  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  protected readonly adding = signal(false);
  protected readonly busy = signal(false);

  protected readonly leads = httpResource<CrmLead[]>(() => '/crm/leads');
  protected readonly all = computed(() => loaded(this.leads) ?? []);
  protected readonly columns = computed(() =>
    LEAD_STAGES.map((stage) => ({ stage, items: this.all().filter((l) => l.stage === stage) })),
  );
  private readonly clientList = httpResource<Paginated<CrmClient>>(() =>
    this.adding() ? { url: '/crm/clients', params: { limit: 200 } } : undefined,
  );
  protected readonly clients = computed(() => loaded(this.clientList)?.items ?? []);
  private readonly tourList = httpResource<Tour[]>(() =>
    this.adding() ? '/tours/mine' : undefined,
  );
  protected readonly tours = computed(() => loaded(this.tourList) ?? []);

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.leads'),
      description: this.transloco.translate('crm.leads.subtitle'),
      robots: 'noindex',
    }));
  }

  protected name(c: CrmClient): string {
    return clientName(c);
  }

  protected changed(id: string, lead: CrmLead | null): void {
    this.leads.update((list) =>
      (list ?? []).flatMap((l) => (l.id !== id ? [l] : lead ? [lead] : [])),
    );
  }

  protected async create(
    event: Event,
    clientId: string,
    tourId: string,
    note: string,
  ): Promise<void> {
    event.preventDefault();
    if (!clientId) return;
    this.busy.set(true);
    try {
      const lead = await firstValueFrom(
        this.http.post<CrmLead>('/crm/leads', {
          clientId,
          ...(tourId ? { tourId } : {}),
          note: note.trim() || null,
        }),
      );
      this.leads.update((list) => [lead, ...(list ?? [])]);
      this.adding.set(false);
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }
}
