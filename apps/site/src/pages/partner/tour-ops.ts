import type { CrmGroup, ExpenseCategory, StaffRole } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { Notify } from '@/core/notify/notify';
import { injectConfirm } from '@/ui/confirm-dialog';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';

const CATEGORIES: ExpenseCategory[] = [
  'transport',
  'guides',
  'food',
  'accommodation',
  'permits',
  'gear',
  'other',
];
const ROLES: StaffRole[] = ['guide', 'assistant', 'driver', 'cook', 'other'];

// A tour's money and people next to its group: what was paid, the
// expenses by category and the margin; who leads and drives the group.
// Every change answers with the whole group, handed up through `changed`.
@Component({
  selector: 'app-tour-ops',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'grid gap-4 lg:grid-cols-2' },
  template: `
    @let g = group();
    <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="tour-finance">
      <h2 id="tour-finance" class="font-semibold text-text">{{ 'crm.finance.title' | transloco }}</h2>
      <dl class="grid grid-cols-3 gap-2 text-sm">
        <div><dt class="text-xs text-text-muted">{{ 'crm.paid' | transloco }}</dt><dd class="font-bold text-text tabular-nums">{{ amount(g.totals.paid) }}</dd></div>
        <div><dt class="text-xs text-text-muted">{{ 'crm.finance.expenses' | transloco }}</dt><dd class="font-bold text-text tabular-nums">{{ amount(g.totals.expenses) }}</dd></div>
        <div><dt class="text-xs text-text-muted">{{ 'crm.finance.margin' | transloco }}</dt><dd class="font-bold tabular-nums" [class]="g.totals.margin < 0 ? 'text-danger' : 'text-success'">{{ amount(g.totals.margin) }}</dd></div>
      </dl>
      @if (g.expenses.length > 0) {
        <ul class="flex flex-col divide-y divide-border text-sm">
          @for (e of g.expenses; track e.id) {
            <li class="flex items-center gap-2 py-1.5">
              <span class="text-text">{{ 'crm.expense.' + e.category | transloco }}</span>
              @if (e.note) {
                <span class="truncate text-text-muted">— {{ e.note }}</span>
              }
              <span class="ml-auto font-medium text-text tabular-nums">{{ amount(e.amount) }}</span>
              <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.finance.delete' | transloco" [disabled]="busy()" (click)="remove('/crm/expenses/' + e.id)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
            </li>
          }
        </ul>
      }
      <form class="grid gap-2 sm:grid-cols-[9rem_7rem_1fr_auto] sm:items-end" novalidate (submit)="addExpense($event, category.value, expenseAmount.value, expenseNote.value)">
        <div class="ui-field">
          <label class="ui-label" for="expense-category">{{ 'crm.finance.category' | transloco }}</label>
          <select #category id="expense-category" class="ui-input">
            @for (c of categories; track c) {
              <option [value]="c">{{ 'crm.expense.' + c | transloco }}</option>
            }
          </select>
        </div>
        <div class="ui-field">
          <label class="ui-label" for="expense-amount">{{ 'crm.payment.amount' | transloco }}</label>
          <input #expenseAmount id="expense-amount" class="ui-input" type="number" min="1" inputmode="numeric" />
        </div>
        <div class="ui-field">
          <label class="ui-label" for="expense-note">{{ 'crm.payment.note' | transloco }}</label>
          <input #expenseNote id="expense-note" class="ui-input" maxlength="500" />
        </div>
        <button type="submit" class="ui-btn-soft" [disabled]="busy()">{{ 'crm.finance.add' | transloco }}</button>
      </form>
    </section>

    <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="tour-staff">
      <h2 id="tour-staff" class="font-semibold text-text">{{ 'crm.staff.title' | transloco }}</h2>
      @if (g.staff.length > 0) {
        <ul class="flex flex-col divide-y divide-border text-sm">
          @for (m of g.staff; track m.id) {
            <li class="flex items-center gap-2 py-1.5">
              <span class="ui-badge ui-badge-muted">{{ 'crm.staff.' + m.role | transloco }}</span>
              <span class="font-medium text-text">{{ m.name }}</span>
              @if (m.phone) {
                <a class="text-text-muted hover:underline" [href]="'tel:' + m.phone">{{ m.phone }}</a>
              }
              @if (m.note) {
                <span class="truncate text-text-muted">— {{ m.note }}</span>
              }
              <button type="button" class="ui-icon-btn ml-auto size-7" [attr.aria-label]="'crm.staff.delete' | transloco" [disabled]="busy()" (click)="remove('/crm/staff/' + m.id)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
            </li>
          }
        </ul>
      } @else {
        <p class="text-sm text-text-muted">{{ 'crm.staff.empty' | transloco }}</p>
      }
      <form class="grid gap-2 sm:grid-cols-[8rem_1fr_9rem_auto] sm:items-end" novalidate (submit)="addStaff($event, role.value, staffName.value, staffPhone.value)">
        <div class="ui-field">
          <label class="ui-label" for="staff-role">{{ 'crm.staff.role' | transloco }}</label>
          <select #role id="staff-role" class="ui-input">
            @for (r of roles; track r) {
              <option [value]="r">{{ 'crm.staff.' + r | transloco }}</option>
            }
          </select>
        </div>
        <div class="ui-field">
          <label class="ui-label" for="staff-name">{{ 'crm.client.name' | transloco }}</label>
          <input #staffName id="staff-name" class="ui-input" maxlength="120" />
        </div>
        <div class="ui-field">
          <label class="ui-label" for="staff-phone">{{ 'crm.client.phone' | transloco }}</label>
          <input #staffPhone id="staff-phone" class="ui-input" type="tel" />
        </div>
        <button type="submit" class="ui-btn-soft" [disabled]="busy()">{{ 'crm.staff.add' | transloco }}</button>
      </form>
    </section>
  `,
})
export class TourOps {
  readonly group = input.required<CrmGroup>();
  readonly changed = output<CrmGroup>();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly categories = CATEGORIES;
  protected readonly roles = ROLES;
  protected readonly busy = signal(false);

  protected amount(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected addExpense(event: Event, category: string, rawAmount: string, note: string): void {
    event.preventDefault();
    const amount = Math.round(Number(rawAmount));
    if (!(amount > 0)) {
      this.notify.warning({ title: 'crm.errors.amount' });
      return;
    }
    void this.run(
      this.http.post<CrmGroup>(`/crm/tours/${this.group().tour.id}/expenses`, {
        category,
        amount,
        note: note.trim() || null,
      }),
      event,
    );
  }

  protected addStaff(event: Event, role: string, name: string, phone: string): void {
    event.preventDefault();
    if (!name.trim()) {
      this.notify.warning({ title: 'crm.errors.nameRequired' });
      return;
    }
    void this.run(
      this.http.post<CrmGroup>(`/crm/tours/${this.group().tour.id}/staff`, {
        role,
        name: name.trim(),
        phone: phone.trim() || null,
      }),
      event,
    );
  }

  protected async remove(url: string): Promise<void> {
    if (!(await this.confirm({ title: 'crm.finance.deleteConfirm', danger: true }))) return;
    await this.run(this.http.delete<CrmGroup>(url));
  }

  private async run(request: Observable<CrmGroup>, event?: Event): Promise<void> {
    this.busy.set(true);
    try {
      this.changed.emit(await firstValueFrom(request));
      (event?.target as HTMLFormElement | undefined)?.reset();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }
}
