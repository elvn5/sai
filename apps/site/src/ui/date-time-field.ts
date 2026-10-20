import { injectLang } from '@/core/i18n/current-lang';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  Injector,
  afterNextRender,
  computed,
  inject,
  input,
  model,
  output,
  signal,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe } from '@jsverse/transloco';

// Wall-clock parts of a "YYYY-MM-DD" or "YYYY-MM-DDTHH:mm" value. The field
// knows nothing of time zones: the value is the time as the visitor reads it.
export interface DateParts {
  year: number;
  month: number; // 0–11
  day: number;
  hour: number;
  minute: number;
}

export function parseValue(value: string): DateParts | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?$/.exec(value);
  if (!match) return null;
  const [, y, m, d, hh, mm] = match;
  return { year: +y, month: +m - 1, day: +d, hour: hh ? +hh : 0, minute: mm ? +mm : 0 };
}

const pad = (n: number): string => String(n).padStart(2, '0');

// "октябрь 2026 г." → "Октябрь 2026 г." (CSS capitalize would make it "Г.").
const upperFirst = (text: string): string => text.charAt(0).toLocaleUpperCase() + text.slice(1);

export function dateKey(year: number, month: number, day: number): string {
  // Normalize overflow (day 0, month 12…) through UTC.
  const date = new Date(Date.UTC(year, month, day));
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`;
}

export function formatValue(parts: DateParts, withTime: boolean): string {
  const date = dateKey(parts.year, parts.month, parts.day);
  return withTime ? `${date}T${pad(parts.hour)}:${pad(parts.minute)}` : date;
}

// The parts of a "YYYY-MM-DD" key the calendar made itself.
export function keyParts(key: string): { year: number; month: number; day: number } {
  const [year, month, day] = key.split('-').map(Number);
  return { year, month: month - 1, day };
}

export interface CalendarDay {
  key: string;
  day: number;
  inMonth: boolean;
}

// Six weeks from the Monday on or before the 1st of the month.
export function calendarWeeks(year: number, month: number): CalendarDay[][] {
  const first = new Date(Date.UTC(year, month, 1));
  const offset = (first.getUTCDay() + 6) % 7;
  const weeks: CalendarDay[][] = [];
  for (let w = 0; w < 6; w++) {
    const week: CalendarDay[] = [];
    for (let d = 0; d < 7; d++) {
      const date = new Date(Date.UTC(year, month, 1 - offset + w * 7 + d));
      week.push({
        key: dateKey(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()),
        day: date.getUTCDate(),
        inMonth: date.getUTCMonth() === month,
      });
    }
    weeks.push(week);
  }
  return weeks;
}

// Today in the visitor's calendar, as "YYYY-MM-DD".
function todayKey(): string {
  const now = new Date();
  return dateKey(now.getFullYear(), now.getMonth(), now.getDate());
}

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 12 }, (_, i) => i * 5);

let nextId = 0;

// A date (and time) picker: the field shows the value in words, a click
// opens a calendar with hour and minute columns. Keyboard: arrows move by
// a day or a week, Page Up/Down by a month, Home/End to the week's ends,
// Enter picks, Escape closes.
@Component({
  selector: 'app-date-time-field',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'relative block',
    '(document:click)': 'onDocumentClick($event)',
    '(focusout)': 'onFocusOut($event)',
    '(keydown.escape)': 'close(true)',
  },
  template: `
    <button
      type="button"
      class="ui-input flex items-center gap-2 text-left"
      [class.text-text-muted]="!parts()"
      aria-haspopup="dialog"
      [attr.aria-expanded]="open()"
      [attr.aria-controls]="panelId"
      [attr.aria-labelledby]="labelledBy() ? labelledBy() + ' ' + triggerId : null"
      [attr.aria-invalid]="invalid() || null"
      [id]="triggerId"
      [disabled]="disabled()"
      (click)="toggle()"
    >
      <span class="icon-[heroicons--calendar-days] size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
      <span class="min-w-0 flex-1 truncate">{{ display() || ((withTime() ? 'picker.chooseDateTime' : 'picker.chooseDate') | transloco) }}</span>
      <span class="icon-[heroicons--chevron-down] size-4 shrink-0 text-text-muted transition-transform" [class.rotate-180]="open()" aria-hidden="true"></span>
    </button>

    @if (open()) {
      <div
        role="dialog"
        [id]="panelId"
        [attr.aria-label]="(withTime() ? 'picker.chooseDateTime' : 'picker.chooseDate') | transloco"
        class="motion-drop-in absolute top-full z-50 mt-2 w-max max-w-[calc(100vw-2rem)] rounded-xl border border-border bg-surface p-3 text-sm text-text shadow-lg"
        [class.right-0]="anchor() === 'end'"
        [class.left-0]="anchor() !== 'end'"
      >
        <div class="flex flex-col gap-3 sm:flex-row">
          <div class="w-[17.5rem] max-w-full">
            <div class="mb-2 flex items-center justify-between gap-2">
              <button type="button" class="ui-picker-nav" [attr.aria-label]="'picker.prevMonth' | transloco" (click)="shiftMonth(-1)">
                <span class="icon-[heroicons--chevron-left] size-4" aria-hidden="true"></span>
              </button>
              <span class="font-semibold" aria-live="polite">{{ monthTitle() }}</span>
              <button type="button" class="ui-picker-nav" [attr.aria-label]="'picker.nextMonth' | transloco" (click)="shiftMonth(1)">
                <span class="icon-[heroicons--chevron-right] size-4" aria-hidden="true"></span>
              </button>
            </div>
            <div class="grid grid-cols-7 gap-0.5 text-center">
              @for (name of weekdays(); track $index) {
                <span class="pb-1 text-xs font-medium text-text-muted" aria-hidden="true">{{ name }}</span>
              }
              @for (week of weeks(); track $index) {
                @for (cell of week; track cell.key) {
                  <button
                    type="button"
                    class="ui-picker-day"
                    [class.text-text-muted]="!cell.inMonth"
                    [attr.data-date]="cell.key"
                    [attr.aria-label]="dayLabel(cell.key)"
                    [attr.aria-pressed]="cell.key === selectedKey()"
                    [attr.aria-current]="cell.key === today ? 'date' : null"
                    [disabled]="isBeforeMin(cell.key)"
                    [tabindex]="cell.key === activeKey() ? 0 : -1"
                    (click)="pickDay(cell.key)"
                    (keydown)="onGridKeydown($event)"
                  >
                    {{ cell.day }}
                  </button>
                }
              }
            </div>
          </div>

          @if (withTime()) {
            <div class="flex flex-col gap-2 border-t border-border pt-3 sm:w-32 sm:border-t-0 sm:border-l sm:pt-0 sm:pl-3">
              <div class="flex flex-col gap-2 sm:grid sm:grid-cols-2 sm:gap-1">
                <div class="flex min-w-0 items-center gap-2 sm:flex-col sm:items-stretch sm:gap-1">
                  <span class="w-12 shrink-0 text-xs font-medium text-text-muted sm:w-auto sm:text-center" aria-hidden="true">{{ 'picker.hours' | transloco }}</span>
                  <div class="ui-picker-times" role="group" [attr.aria-label]="'picker.hours' | transloco">
                    @for (hour of hours; track hour) {
                      <button type="button" class="ui-picker-time" [attr.aria-pressed]="parts()?.hour === hour" (click)="pickTime(hour, null)">{{ pad(hour) }}</button>
                    }
                  </div>
                </div>
                <div class="flex min-w-0 items-center gap-2 sm:flex-col sm:items-stretch sm:gap-1">
                  <span class="w-12 shrink-0 text-xs font-medium text-text-muted sm:w-auto sm:text-center" aria-hidden="true">{{ 'picker.minutes' | transloco }}</span>
                  <div class="ui-picker-times" role="group" [attr.aria-label]="'picker.minutes' | transloco">
                    @for (minute of minutes; track minute) {
                      <button type="button" class="ui-picker-time" [attr.aria-pressed]="parts()?.minute === minute" (click)="pickTime(null, minute)">{{ pad(minute) }}</button>
                    }
                  </div>
                </div>
              </div>
            </div>
          }
        </div>

        @if (clearable() || withTime()) {
          <div class="mt-3 flex items-center justify-between gap-2 border-t border-border pt-3">
            @if (clearable() && value()) {
              <button type="button" class="ui-btn-ghost px-2 py-1.5 text-sm" (click)="clear()">{{ 'picker.clear' | transloco }}</button>
            } @else {
              <span></span>
            }
            @if (withTime()) {
              <button type="button" class="ui-btn-primary px-4 py-1.5 text-sm" (click)="close(true)">{{ 'picker.done' | transloco }}</button>
            }
          </div>
        }
      </div>
    }
  `,
})
export class DateTimeField implements FormValueControl<string> {
  readonly value = model('');
  readonly withTime = input(true);
  // The earliest day that can be picked, "YYYY-MM-DD".
  readonly earliest = input<string | null>(null);
  // Time given to a day picked while the value has none.
  readonly defaultTime = input('09:00');
  readonly clearable = input(false);
  // The panel's edge that lines up with the field ("end" for right columns).
  readonly anchor = input<'start' | 'end'>('start');
  // Id of the field's visible label.
  readonly labelledBy = input<string | null>(null);
  readonly disabled = input(false);
  readonly invalid = input(false);
  readonly touch = output<void>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);
  private readonly lang = injectLang();

  protected readonly panelId = `date-time-panel-${nextId}`;
  protected readonly triggerId = `date-time-${nextId++}`;
  protected readonly hours = HOURS;
  protected readonly minutes = MINUTES;
  protected readonly pad = pad;
  protected readonly today = todayKey();

  protected readonly open = signal(false);
  // The month on screen and the day keyboard focus sits on.
  private readonly view = signal({ year: 2000, month: 0 });
  protected readonly activeKey = signal('');

  protected readonly parts = computed(() => parseValue(this.value()));
  protected readonly selectedKey = computed(() => {
    const p = this.parts();
    return p ? dateKey(p.year, p.month, p.day) : null;
  });
  protected readonly weeks = computed(() => calendarWeeks(this.view().year, this.view().month));

  // "сб, 24 октября, 06:00": the year only when it isn't this one.
  protected readonly display = computed(() => {
    const p = this.parts();
    if (!p) return '';
    const day = new Intl.DateTimeFormat(this.lang(), {
      weekday: 'short',
      day: 'numeric',
      month: 'long',
      ...(p.year === keyParts(this.today).year ? {} : { year: 'numeric' }),
      timeZone: 'UTC',
    }).format(new Date(Date.UTC(p.year, p.month, p.day)));
    return this.withTime() ? `${day}, ${pad(p.hour)}:${pad(p.minute)}` : day;
  });
  protected readonly monthTitle = computed(() => {
    const { year, month } = this.view();
    return upperFirst(
      new Intl.DateTimeFormat(this.lang(), {
        month: 'long',
        year: 'numeric',
        timeZone: 'UTC',
      }).format(new Date(Date.UTC(year, month, 1))),
    );
  });
  // Monday first; 2024-01-01 was a Monday.
  protected readonly weekdays = computed(() => {
    const format = new Intl.DateTimeFormat(this.lang(), { weekday: 'short', timeZone: 'UTC' });
    return Array.from({ length: 7 }, (_, i) =>
      upperFirst(format.format(new Date(Date.UTC(2024, 0, 1 + i)))),
    );
  });

  protected dayLabel(key: string): string {
    const p = parseValue(key);
    if (!p) return key;
    return new Intl.DateTimeFormat(this.lang(), { dateStyle: 'full', timeZone: 'UTC' }).format(
      new Date(Date.UTC(p.year, p.month, p.day)),
    );
  }

  protected isBeforeMin(key: string): boolean {
    const min = this.earliest();
    return !!min && key < min;
  }

  protected toggle(): void {
    if (this.open()) {
      this.close(false);
      return;
    }
    const earliest = this.earliest();
    const start = this.selectedKey() ?? (earliest && earliest > this.today ? earliest : this.today);
    const p = keyParts(start);
    this.view.set({ year: p.year, month: p.month });
    this.activeKey.set(start);
    this.open.set(true);
    afterNextRender(() => this.scrollTimeIntoView(), { injector: this.injector });
  }

  close(returnFocus: boolean): void {
    if (!this.open()) return;
    this.open.set(false);
    this.touch.emit();
    if (returnFocus)
      this.host.nativeElement.querySelector<HTMLElement>(`#${this.triggerId}`)?.focus();
  }

  protected shiftMonth(delta: number): void {
    const { year, month } = this.view();
    const p = keyParts(dateKey(year, month + delta, 1));
    this.view.set({ year: p.year, month: p.month });
    // Keep the keyboard on the same day of the new month (or its last).
    const active = parseValue(this.activeKey());
    const lastDay = new Date(Date.UTC(p.year, p.month + 1, 0)).getUTCDate();
    this.activeKey.set(dateKey(p.year, p.month, Math.min(active?.day ?? 1, lastDay)));
  }

  protected pickDay(key: string): void {
    if (this.isBeforeMin(key)) return;
    const day = keyParts(key);
    if (!this.withTime()) {
      this.value.set(key);
      this.close(true);
      return;
    }
    const current = this.parts();
    const fallback = parseValue(`2000-01-01T${this.defaultTime()}`);
    this.value.set(
      formatValue(
        {
          ...day,
          hour: current?.hour ?? fallback?.hour ?? 9,
          minute: current?.minute ?? fallback?.minute ?? 0,
        },
        true,
      ),
    );
    this.activeKey.set(key);
    afterNextRender(() => this.scrollTimeIntoView(), { injector: this.injector });
  }

  protected pickTime(hour: number | null, minute: number | null): void {
    const current = this.parts() ?? parseValue(`${this.activeKey()}T${this.defaultTime()}`);
    if (!current) return;
    this.value.set(
      formatValue(
        { ...current, hour: hour ?? current.hour, minute: minute ?? current.minute },
        true,
      ),
    );
  }

  protected clear(): void {
    this.value.set('');
    this.close(true);
  }

  protected onGridKeydown(event: KeyboardEvent): void {
    const active = parseValue(this.activeKey());
    if (!active) return;
    const weekday = (new Date(Date.UTC(active.year, active.month, active.day)).getUTCDay() + 6) % 7;
    const steps: Record<string, () => string> = {
      ArrowLeft: () => dateKey(active.year, active.month, active.day - 1),
      ArrowRight: () => dateKey(active.year, active.month, active.day + 1),
      ArrowUp: () => dateKey(active.year, active.month, active.day - 7),
      ArrowDown: () => dateKey(active.year, active.month, active.day + 7),
      Home: () => dateKey(active.year, active.month, active.day - weekday),
      End: () => dateKey(active.year, active.month, active.day + 6 - weekday),
      PageUp: () => dateKey(active.year, active.month - 1, active.day),
      PageDown: () => dateKey(active.year, active.month + 1, active.day),
    };
    const step = steps[event.key];
    if (!step) return;
    event.preventDefault();
    const next = step();
    const p = keyParts(next);
    this.activeKey.set(next);
    this.view.set({ year: p.year, month: p.month });
    afterNextRender(
      () => this.host.nativeElement.querySelector<HTMLElement>(`[data-date="${next}"]`)?.focus(),
      { injector: this.injector },
    );
  }

  protected onDocumentClick(event: MouseEvent): void {
    if (this.open() && !this.host.nativeElement.contains(event.target as Node)) this.close(false);
  }

  protected onFocusOut(event: FocusEvent): void {
    const next = event.relatedTarget as Node | null;
    if (next && !this.host.nativeElement.contains(next)) this.close(false);
  }

  // The picked hour and minute in the middle of their columns.
  private scrollTimeIntoView(): void {
    for (const list of this.host.nativeElement.querySelectorAll<HTMLElement>('[role="group"]')) {
      const pressed = list.querySelector<HTMLElement>('[aria-pressed="true"]');
      if (!pressed) continue;
      list.scrollTop =
        pressed.offsetTop - list.offsetTop - list.clientHeight / 2 + pressed.offsetHeight / 2;
      list.scrollLeft =
        pressed.offsetLeft - list.offsetLeft - list.clientWidth / 2 + pressed.offsetWidth / 2;
    }
  }
}
