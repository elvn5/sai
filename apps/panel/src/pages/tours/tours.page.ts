import type { AdminTourList, AdminTourRow, TourPhase } from '@/core/api/types';
import { personName, som } from '@/core/lib/format';
import {
  PHASES,
  meetingTime,
  phaseLabel,
  phaseTone,
  places,
  tourDates,
} from '@/pages/tours/tour-labels';
import { HistoryLog } from '@/ui/history-log';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

// Tours: every departure with its group — on the way now, ahead and over —
// to follow sign-ups, the waitlist, check-in, forms and payments.
@Component({
  selector: 'app-tours-page',
  imports: [HistoryLog, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div>
        <h1 class="text-xl font-semibold text-text">Туры</h1>
        <p class="text-sm text-text-muted">
          Все выезды организаторов и их участники. «Сейчас идут» — группа уже собралась, тур ещё не закончился
          (до 18:00 последнего дня по Бишкеку).
        </p>
        @if (list.error()) {
          <p class="ui-error mt-2">Не удалось загрузить туры</p>
        }
      </div>

      <div class="flex flex-wrap items-end gap-3">
        <div class="flex flex-wrap gap-1" role="tablist" aria-label="Период">
          @for (p of phases; track p.value) {
            <button
              type="button"
              role="tab"
              [class]="phase() === p.value ? 'ui-btn-primary' : 'ui-btn-soft'"
              [attr.aria-selected]="phase() === p.value"
              (click)="phase.set(p.value)"
            >
              {{ p.label }} <span class="opacity-70">{{ counts()?.[p.value] ?? 0 }}</span>
            </button>
          }
          <button
            type="button"
            role="tab"
            [class]="phase() === '' ? 'ui-btn-primary' : 'ui-btn-soft'"
            [attr.aria-selected]="phase() === ''"
            (click)="phase.set('')"
          >
            Все <span class="opacity-70">{{ total() }}</span>
          </button>
        </div>
        <label class="ui-field">
          <span>Поиск</span>
          <input
            class="ui-input w-64"
            type="search"
            placeholder="Тур, место, организатор"
            [value]="query()"
            (input)="query.set($any($event.target).value)"
          />
        </label>
      </div>

      @if (rows().length) {
        <p class="text-sm text-text-muted">
          {{ rows().length }} {{ toursWord(rows().length) }} · участников с местом: {{ summary().active }}
          · в листе ожидания: {{ summary().waitlist }}
          @if (phase() === 'current') {
            · на месте сбора отмечено: {{ summary().arrived }}
          }
        </p>
      }

      <div class="ui-table-wrap">
        <table class="ui-table">
          <thead>
            <tr>
              <th>Тур</th><th>Даты и сбор</th><th>Организатор</th><th>Места</th><th>Пришли</th><th>Анкеты</th><th>Оплачено</th>
            </tr>
          </thead>
          <tbody>
            @for (row of rows(); track row.tour.id) {
              <tr>
                <td>
                  <a class="font-medium text-text underline-offset-2 hover:underline" [routerLink]="['/tours', row.tour.id]">{{ row.tour.title }}</a>
                  <p class="mt-1 flex flex-wrap gap-1">
                    <span class="ui-badge" [class]="phaseTone(row.phase)">{{ phaseLabel(row.phase) }}</span>
                    @if (!row.tour.isPublished) {
                      <span class="ui-badge bg-red-100 text-red-800">скрыт</span>
                    }
                    <span class="ui-badge bg-gray-100 text-gray-600">{{ row.tour.locale }}</span>
                  </p>
                </td>
                <td>
                  {{ tourDates(row.tour) }}
                  <p class="text-xs text-text-muted">
                    сбор в {{ meetingTime(row.tour) }}@if (row.tour.meetingPoint) {, {{ row.tour.meetingPoint }}}
                  </p>
                </td>
                <td>{{ personName(row.tour.organizer) }}</td>
                <td>
                  <span class="font-medium" [class.text-red-700]="isFull(row)">{{ places(row.counts.active, row.tour.capacity) }}</span>
                  @if (row.counts.waitlist) {
                    <p class="text-xs text-amber-700">+{{ row.counts.waitlist }} в ожидании</p>
                  }
                  @if (row.counts.cancelled) {
                    <p class="text-xs text-text-muted">{{ row.counts.cancelled }} отменили</p>
                  }
                </td>
                <td>{{ row.phase === 'upcoming' ? '—' : row.counts.arrived + ' из ' + row.counts.active }}</td>
                <td>
                  {{ row.counts.forms }} из {{ row.counts.active }}
                  <p class="text-xs text-text-muted">правила: {{ row.counts.waivers }}</p>
                </td>
                <td>
                  {{ som(row.counts.paid) }}
                  @if (row.counts.due) {
                    <p class="text-xs text-text-muted">из {{ som(row.counts.due) }}</p>
                  }
                </td>
              </tr>
            } @empty {
              @if (!list.isLoading()) {
                <tr><td colspan="7" class="ui-empty">{{ emptyText() }}</td></tr>
              }
            }
          </tbody>
        </table>
      </div>
    </div>

    <app-history-log class="mt-8" [modules]="['tours']" />
  `,
})
export class ToursPage {
  protected readonly phases = PHASES;
  // The tours on the way first: what the panel lacked.
  protected readonly phase = signal<TourPhase | ''>('current');
  protected readonly query = signal('');

  protected readonly list = httpResource<AdminTourList>(() => {
    const query = new URLSearchParams();
    if (this.phase()) query.set('phase', this.phase());
    const q = this.query().trim();
    if (q) query.set('q', q);
    const qs = query.toString();
    return `/tours/admin${qs ? `?${qs}` : ''}`;
  });
  protected readonly rows = computed(() => this.list.value()?.tours ?? []);
  protected readonly counts = computed(() => this.list.value()?.counts);
  protected readonly total = computed(() => {
    const c = this.counts();
    return c ? c.current + c.upcoming + c.past : 0;
  });
  protected readonly summary = computed(() =>
    this.rows().reduce(
      (sum, row) => ({
        active: sum.active + row.counts.active,
        waitlist: sum.waitlist + row.counts.waitlist,
        arrived: sum.arrived + row.counts.arrived,
      }),
      { active: 0, waitlist: 0, arrived: 0 },
    ),
  );
  protected readonly emptyText = computed(() => {
    if (this.query().trim()) return 'Ничего не нашлось';
    switch (this.phase()) {
      case 'current':
        return 'Сейчас ни один тур не идёт';
      case 'upcoming':
        return 'Предстоящих туров нет';
      case 'past':
        return 'Прошедших туров нет';
      default:
        return 'Туров пока нет';
    }
  });

  protected readonly som = som;
  protected readonly personName = personName;
  protected readonly phaseLabel = phaseLabel;
  protected readonly phaseTone = phaseTone;
  protected readonly places = places;
  protected readonly tourDates = tourDates;
  protected readonly meetingTime = meetingTime;

  protected isFull(row: AdminTourRow): boolean {
    return row.tour.capacity !== null && row.counts.active >= row.tour.capacity;
  }

  protected toursWord(n: number): string {
    const mod10 = n % 10;
    const mod100 = n % 100;
    if (mod10 === 1 && mod100 !== 11) return 'тур';
    if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'тура';
    return 'туров';
  }
}
