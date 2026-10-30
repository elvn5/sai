import type {
  CrmGroup,
  CrmParticipant,
  ParticipantForm,
  ParticipantStatus,
  PaymentMethod,
} from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, todayInBishkek } from '@/core/lib/format';
import { Notify } from '@/core/notify/notify';
import { PARTICIPANT_STATUSES, participantName, statusBadge } from '@/pages/partner/crm';
import { injectConfirm } from '@/ui/confirm-dialog';
import { FieldErrors } from '@/ui/field-errors';
import { ParticipantFormView } from '@/ui/participant-form';
import { PersonAvatar } from '@/ui/person-avatar';
import { SelectField, injectOptions } from '@/ui/select-field';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const METHODS: PaymentMethod[] = ['cash', 'transfer', 'card', 'other'];

// One participant of a tour's group in the CRM: status, check-in, what
// they owe, and their payments (shown and added when opened). Every change
// answers with the whole group, handed up through `changed`.
@Component({
  selector: 'app-participant-row',
  imports: [
    FieldErrors,
    FormField,
    ParticipantFormView,
    PersonAvatar,
    SelectField,
    SiteLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @let p = participant();
    <div class="flex flex-col gap-3 px-4 py-3" [class.opacity-60]="p.status === 'cancelled'">
      <div class="flex flex-wrap items-center gap-3">
        <span class="w-6 shrink-0 text-right text-sm text-text-muted tabular-nums">{{ index() }}</span>
        @if (p.account) {
          <app-person-avatar [person]="p.account" [size]="36" />
        } @else {
          <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-text/7 text-text-muted"><span class="icon-[heroicons--user] size-4.5" aria-hidden="true"></span></span>
        }
        <div class="min-w-0 flex-1">
          <a [appSiteLink]="'/partner/clients/' + p.clientId" class="block truncate font-medium text-text hover:underline">{{ name() }}</a>
          <p class="flex flex-wrap items-center gap-x-2 text-xs text-text-muted">
            <span class="truncate">{{ contacts() }}</span>
            <span class="inline-flex items-center gap-0.5" [class]="p.form ? 'text-success' : 'text-amber-700 dark:text-amber-300'">
              <span [class]="(p.form ? 'icon-[heroicons--clipboard-document-check]' : 'icon-[heroicons--clipboard-document]') + ' size-3.5'" aria-hidden="true"></span>{{ (p.form ? 'crm.form.filled' : 'crm.form.missing') | transloco }}
            </span>
            @if (p.userId) {
              <span class="inline-flex items-center gap-0.5" [class]="p.waiverAt ? 'text-success' : 'text-amber-700 dark:text-amber-300'">
                <span [class]="(p.waiverAt ? 'icon-[heroicons--shield-check]' : 'icon-[heroicons--shield-exclamation]') + ' size-3.5'" aria-hidden="true"></span>{{ (p.waiverAt ? 'crm.form.waiver' : 'crm.form.noWaiver') | transloco }}
              </span>
            }
          </p>
        </div>
        <span class="sr-only" [id]="'status-' + p.id">{{ 'crm.statusLabel' | transloco }}</span>
        <app-select-field class="w-40" size="sm" [options]="statusOptions()" [labelledBy]="'status-' + p.id" [value]="p.status" [disabled]="busy()" (valueChange)="setStatus($any($event))" />
        @if (p.status === 'new' || p.status === 'confirmed') {
          <button type="button" class="ui-btn-soft ui-btn-xs" [attr.aria-pressed]="!!p.arrivedAt" [disabled]="busy()" (click)="toggleArrived()">
            <span [class]="(p.arrivedAt ? 'icon-[heroicons--check-circle-solid] text-success' : 'icon-[heroicons--minus-circle] text-text-muted') + ' size-4'" aria-hidden="true"></span>{{ 'crm.arrived' | transloco }}
          </button>
        }
        <button type="button" class="ui-btn-ghost ui-btn-xs tabular-nums" [attr.aria-expanded]="open()" (click)="open.set(!open())">
          <span [class]="debt() > 0 ? 'text-amber-700 dark:text-amber-300' : 'text-text'">{{ amount(p.paid) }} / {{ amount(p.due) }}</span>
          <span [class]="(open() ? 'icon-[heroicons--chevron-up]' : 'icon-[heroicons--chevron-down]') + ' size-4'" aria-hidden="true"></span>
          <span class="sr-only">{{ 'crm.payments' | transloco }}</span>
        </button>
      </div>

      @if (open()) {
        <div class="ml-9 flex flex-col gap-3 rounded-lg bg-text/3 p-3">
          <div class="flex flex-wrap items-end gap-3 text-sm">
            <div class="ui-field w-36">
              <label class="ui-label" [for]="'due-' + p.id">{{ 'crm.due' | transloco }}</label>
              <input class="ui-input" type="number" min="0" inputmode="numeric" [id]="'due-' + p.id" [value]="p.amountDue ?? ''" [placeholder]="amount(p.due)" (change)="setDue($any($event.target).value)" />
            </div>
            <p class="pb-2 text-text-muted">
              {{ 'crm.paid' | transloco }}: <b class="text-text">{{ amount(p.paid) }}</b>
              · {{ 'crm.debt' | transloco }}: <b [class]="debt() > 0 ? 'text-amber-700 dark:text-amber-300' : 'text-text'">{{ amount(debt()) }}</b>
            </p>
          </div>

          @if (p.payments.length > 0) {
            <ul class="flex flex-col gap-1 text-sm">
              @for (pay of p.payments; track pay.id) {
                <li class="flex items-center gap-2">
                  <span [class]="(pay.kind === 'refund' ? 'icon-[heroicons--arrow-uturn-left] text-danger' : 'icon-[heroicons--banknotes] text-text-muted') + ' size-4'" aria-hidden="true"></span>
                  <span class="font-medium tabular-nums" [class]="pay.kind === 'refund' ? 'text-danger' : 'text-text'">@if (pay.kind === 'refund') {−}{{ amount(pay.amount) }}</span>
                  @if (pay.kind === 'refund') {
                    <span class="ui-badge ui-badge-danger">{{ 'crm.payment.refund' | transloco }}</span>
                  }
                  <span class="text-text-muted">{{ 'crm.method.' + pay.method | transloco }} · {{ date(pay.paidOn) }}</span>
                  @if (pay.note) {
                    <span class="truncate text-text-muted">— {{ pay.note }}</span>
                  }
                  <button type="button" class="ui-icon-btn ml-auto size-7" [attr.aria-label]="'crm.payment.delete' | transloco" [disabled]="busy()" (click)="deletePayment(pay.id)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
                </li>
              }
            </ul>
          }

          <form class="grid gap-2 sm:grid-cols-[8rem_8rem_9rem_10rem_1fr_auto] sm:items-end" novalidate (submit)="addPayment($event)">
            <div class="ui-field">
              <span class="ui-label" [id]="'kind-' + p.id">{{ 'crm.payment.kind' | transloco }}</span>
              <app-select-field [options]="kindOptions()" [labelledBy]="'kind-' + p.id" [formField]="paymentForm.kind" />
            </div>
            <div class="ui-field">
              <label class="ui-label" [for]="'amount-' + p.id">{{ 'crm.payment.amount' | transloco }}</label>
              <input class="ui-input" type="number" inputmode="numeric" [id]="'amount-' + p.id" [formField]="paymentForm.amount" />
            </div>
            <div class="ui-field">
              <span class="ui-label" [id]="'method-' + p.id">{{ 'crm.payment.method' | transloco }}</span>
              <app-select-field [options]="methodOptions()" [labelledBy]="'method-' + p.id" [formField]="paymentForm.method" />
            </div>
            <div class="ui-field">
              <label class="ui-label" [for]="'paid-on-' + p.id">{{ 'crm.payment.date' | transloco }}</label>
              <input class="ui-input" type="date" [id]="'paid-on-' + p.id" [formField]="paymentForm.paidOn" />
            </div>
            <div class="ui-field">
              <label class="ui-label" [for]="'note-' + p.id">{{ 'crm.payment.note' | transloco }}</label>
              <input class="ui-input" [id]="'note-' + p.id" [formField]="paymentForm.note" />
            </div>
            <button type="submit" class="ui-btn-primary" [disabled]="paymentForm().submitting()">{{ 'crm.payment.add' | transloco }}</button>
            <app-field-errors class="sm:col-span-6" [state]="paymentForm.amount()" />
          </form>

          <div class="flex flex-col gap-2 border-t border-border pt-3">
            <p class="text-sm font-semibold text-text">{{ 'participantForm.title' | transloco }}</p>
            @if (editingForm()) {
              <app-participant-form [value]="p.form" [busy]="busy()" (saved)="saveForm($event)" (cancelled)="editingForm.set(false)" />
            } @else if (p.form; as f) {
              <dl class="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-2">
                <div><dt class="text-xs text-text-muted">{{ 'participantForm.emergencyName' | transloco }}</dt><dd class="text-text">{{ f.emergencyName }} · <a class="hover:underline" [href]="'tel:' + f.emergencyPhone">{{ f.emergencyPhone }}</a></dd></div>
                <div><dt class="text-xs text-text-muted">{{ 'participantForm.experience' | transloco }}</dt><dd class="text-text">{{ 'participantForm.levels.' + f.experience + '.title' | transloco }}</dd></div>
                @if (f.health) {
                  <div class="sm:col-span-2"><dt class="text-xs text-text-muted">{{ 'participantForm.health' | transloco }}</dt><dd class="whitespace-pre-line text-text">{{ f.health }}</dd></div>
                }
                @if (f.notes) {
                  <div class="sm:col-span-2"><dt class="text-xs text-text-muted">{{ 'participantForm.notes' | transloco }}</dt><dd class="whitespace-pre-line text-text">{{ f.notes }}</dd></div>
                }
              </dl>
              <button type="button" class="ui-btn-ghost ui-btn-xs self-start" (click)="editingForm.set(true)"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'participantForm.edit' | transloco }}</button>
            } @else {
              <p class="text-sm text-text-muted">{{ (p.userId ? 'crm.form.askAccount' : 'crm.form.askManual') | transloco }}</p>
              <button type="button" class="ui-btn-soft ui-btn-xs self-start" (click)="editingForm.set(true)">{{ 'participantForm.fill' | transloco }}</button>
            }
          </div>

          @if (!p.userId) {
            <button type="button" class="ui-btn-ghost ui-btn-xs self-start text-danger" [disabled]="busy()" (click)="remove()"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>{{ 'crm.removeFromTour' | transloco }}</button>
          }
        </div>
      }
    </div>
  `,
})
export class ParticipantRow {
  readonly tourId = input.required<string>();
  readonly participant = input.required<CrmParticipant>();
  readonly index = input.required<number>();
  readonly changed = output<CrmGroup>();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly statusOptions = injectOptions(() =>
    PARTICIPANT_STATUSES.map((s) => ({ value: s, key: `crm.status.${s}` })),
  );
  protected readonly kindOptions = injectOptions(() => [
    { value: 'payment', key: 'crm.payment.payment' },
    { value: 'refund', key: 'crm.payment.refund' },
  ]);
  protected readonly methodOptions = injectOptions(() =>
    METHODS.map((m) => ({ value: m, key: `crm.method.${m}` })),
  );
  protected readonly busy = signal(false);
  protected readonly open = signal(false);
  protected readonly editingForm = signal(false);
  protected readonly statusBadge = statusBadge;

  protected readonly name = computed(() => participantName(this.participant()));
  protected readonly debt = computed(() =>
    Math.max(this.participant().due - this.participant().paid, 0),
  );
  protected readonly contacts = computed(() => {
    const p = this.participant();
    const telegram = p.telegram ?? p.account?.username;
    return [p.phone, telegram ? `@${telegram}` : null].filter(Boolean).join(' · ');
  });

  private readonly paymentModel = linkedSignal(() => ({
    kind: 'payment' as 'payment' | 'refund',
    amount: this.debt() || 0,
    method: 'cash' as PaymentMethod,
    paidOn: todayInBishkek(),
    note: '',
  }));
  protected readonly paymentForm = form(this.paymentModel, (path) => {
    validateStandardSchema(
      path,
      z.object({
        kind: z.string(),
        amount: z.number().check(z.gt(0, this.transloco.translate('crm.errors.amount'))),
        method: z.string(),
        paidOn: z.string().check(z.minLength(10, this.transloco.translate('crm.errors.date'))),
        note: z.string().check(z.maxLength(500, this.transloco.translate('crm.errors.tooLong'))),
      }),
    );
  });

  protected amount(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short' });
  }

  private url(suffix = ''): string {
    return `/crm/tours/${this.tourId()}/participants/${this.participant().id}${suffix}`;
  }

  protected setStatus(status: ParticipantStatus): void {
    void this.run(this.http.patch<CrmGroup>(this.url(), { status }));
  }

  protected toggleArrived(): void {
    void this.run(
      this.http.patch<CrmGroup>(this.url(), { arrived: !this.participant().arrivedAt }),
    );
  }

  protected setDue(raw: string): void {
    const value = raw.trim() === '' ? null : Math.max(0, Math.round(Number(raw)));
    if (value !== null && Number.isNaN(value)) return;
    void this.run(this.http.patch<CrmGroup>(this.url(), { amountDue: value }));
  }

  protected async saveForm(form: ParticipantForm): Promise<void> {
    await this.run(this.http.put<CrmGroup>(this.url('/form'), { form }), 'crm.saved');
    this.editingForm.set(false);
  }

  protected async addPayment(event: Event): Promise<void> {
    event.preventDefault();
    await submit(this.paymentForm, async () => {
      const v = this.paymentModel();
      await this.run(
        this.http.post<CrmGroup>(this.url('/payments'), {
          kind: v.kind,
          amount: Math.round(v.amount),
          method: v.method,
          paidOn: v.paidOn,
          note: v.note.trim() || null,
        }),
        v.kind === 'refund' ? 'crm.payment.refunded' : 'crm.payment.added',
      );
      return undefined;
    });
  }

  protected async deletePayment(id: string): Promise<void> {
    if (!(await this.confirm({ title: 'crm.payment.deleteConfirm', danger: true }))) return;
    this.busy.set(true);
    try {
      await firstValueFrom(this.http.delete(`/crm/payments/${id}`));
      this.changed.emit(
        await firstValueFrom(this.http.get<CrmGroup>(`/crm/tours/${this.tourId()}/participants`)),
      );
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(): Promise<void> {
    if (
      !(await this.confirm({
        title: 'crm.removeConfirm',
        params: { name: this.name() },
        danger: true,
      }))
    )
      return;
    await this.run(this.http.delete<CrmGroup>(this.url()));
  }

  // Sends a change; the API answers with the group.
  private async run(request: Observable<CrmGroup>, success?: string): Promise<void> {
    this.busy.set(true);
    try {
      this.changed.emit(await firstValueFrom(request));
      if (success) this.notify.success({ title: success });
    } catch (error) {
      this.notify.error({
        title:
          error instanceof HttpErrorResponse && error.status === 409
            ? 'crm.errors.full'
            : 'crm.saveError',
      });
    } finally {
      this.busy.set(false);
    }
  }
}
