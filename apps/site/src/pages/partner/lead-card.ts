import type { CrmLead, LeadStage } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek } from '@/core/lib/format';
import { Notify } from '@/core/notify/notify';
import { LEAD_STAGES, clientName } from '@/pages/partner/crm';
import { injectConfirm } from '@/ui/confirm-dialog';
import { SelectField, injectOptions } from '@/ui/select-field';
import { SiteLink } from '@/ui/site-link';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// One lead: the client, the tour asked about, the note, and its stage —
// moving it to "lost" asks for the reason. `changed` hands up the saved
// lead (null once deleted).
@Component({
  selector: 'app-lead-card',
  imports: [SelectField, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @let l = lead();
    <article class="ui-card flex flex-col gap-2 p-3 text-sm">
      <div class="flex items-start justify-between gap-2">
        <a [appSiteLink]="'/partner/clients/' + l.client.id" class="font-medium text-text hover:underline">{{ name() }}</a>
        <span class="shrink-0 text-xs text-text-muted">{{ when(l.updatedAt) }}</span>
      </div>
      @if (l.source) {
        <span class="ui-badge ui-badge-muted self-start">{{ 'crm.source.' + l.source | transloco }}</span>
      }
      @if (l.tour) {
        <a [appSiteLink]="'/partner/tours/' + l.tour.id + '/participants'" class="truncate text-text-muted hover:underline">{{ l.tour.title }}</a>
      }
      @if (l.note) {
        <p class="whitespace-pre-line text-text-muted">{{ l.note }}</p>
      }
      @if (l.stage === 'lost' && l.lostReason) {
        <p class="text-danger">{{ 'crm.leads.reason' | transloco }}: {{ l.lostReason }}</p>
      }
      <div class="flex items-center gap-2">
        <span class="sr-only" [id]="'stage-' + l.id">{{ 'crm.leads.stage' | transloco }}</span>
        <app-select-field class="flex-1" size="sm" [options]="stageOptions()" [labelledBy]="'stage-' + l.id" [value]="losing() ? 'lost' : l.stage" [disabled]="busy()" (valueChange)="pick($any($event))" />
        <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.leads.delete' | transloco" [disabled]="busy()" (click)="remove()"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span></button>
      </div>
      @if (losing()) {
        <form class="flex gap-2" (submit)="lose($event, reason.value)">
          <label class="sr-only" [for]="'reason-' + l.id">{{ 'crm.leads.reason' | transloco }}</label>
          <input #reason class="ui-input flex-1 py-1 text-sm" maxlength="300" [id]="'reason-' + l.id" [placeholder]="'crm.leads.reasonPlaceholder' | transloco" />
          <button type="submit" class="ui-btn-soft ui-btn-xs" [disabled]="busy()">{{ 'crm.save' | transloco }}</button>
        </form>
      }
    </article>
  `,
})
export class LeadCard {
  readonly lead = input.required<CrmLead>();
  readonly changed = output<CrmLead | null>();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly stageOptions = injectOptions(() =>
    LEAD_STAGES.map((s) => ({ value: s, key: `crm.leads.stages.${s}` })),
  );
  protected readonly busy = signal(false);
  protected readonly losing = signal(false);

  protected name(): string {
    return clientName(this.lead().client);
  }

  protected when(iso: string): string {
    return formatInBishkek(this.lang(), iso, { day: 'numeric', month: 'short' });
  }

  // Moving to "lost" first asks for the reason (dates, price...).
  protected pick(stage: LeadStage): void {
    this.losing.set(stage === 'lost');
    if (stage !== 'lost') void this.save({ stage });
  }

  protected lose(event: Event, reason: string): void {
    event.preventDefault();
    void this.save({ stage: 'lost', lostReason: reason.trim() || null });
  }

  private async save(change: { stage: LeadStage; lostReason?: string | null }): Promise<void> {
    this.busy.set(true);
    try {
      const lead = await firstValueFrom(
        this.http.patch<CrmLead>(`/crm/leads/${this.lead().id}`, change),
      );
      this.losing.set(false);
      this.changed.emit(lead);
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(): Promise<void> {
    if (!(await this.confirm({ title: 'crm.leads.deleteConfirm', danger: true }))) return;
    this.busy.set(true);
    try {
      await firstValueFrom(this.http.delete(`/crm/leads/${this.lead().id}`));
      this.changed.emit(null);
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }
}
