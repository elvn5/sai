import { apiErrorMessage, hasStatus } from '@/core/api/errors';
import type { AdminTourGroup, AppUser, ParticipantStatus, TourParticipant } from '@/core/api/types';
import { formatDate, formatDateTime, fullName, personName, som } from '@/core/lib/format';
import {
  PARTICIPANT_STATUSES,
  expenseLabel,
  experienceLabel,
  hasPlace,
  meetingTime,
  methodLabel,
  participantName,
  phaseLabel,
  phaseTone,
  places,
  roleLabel,
  statusLabel,
  telegramLink,
  tourDates,
} from '@/pages/tours/tour-labels';
import { HistoryLog } from '@/ui/history-log';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

type Filter = 'all' | 'place' | 'waitlist' | 'cancelled';

const FILTERS: { value: Filter; label: string }[] = [
  { value: 'all', label: 'Все' },
  { value: 'place', label: 'С местом' },
  { value: 'waitlist', label: 'Лист ожидания' },
  { value: 'cancelled', label: 'Отменили' },
];

// One tour: its details and organizer, and its group as the organizer
// sees it — contacts, forms and accepted rules, check-in, payments,
// expenses and staff. The admin can move a participant between a place,
// the waitlist and cancelled, mark them arrived, and hide the tour from
// the catalog.
@Component({
  selector: 'app-tour-page',
  imports: [HistoryLog, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <a routerLink="/tours" class="text-sm text-text-muted hover:text-text">← Туры</a>

      @if (group.error()) {
        <p class="ui-error">{{ hasStatus(group.error(), 404) ? 'Тур не найден' : 'Не удалось загрузить тур' }}</p>
      }
      @if (current(); as g) {
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-semibold text-text">{{ g.tour.title }}</h1>
            <p class="text-sm text-text-muted">
              {{ tourDates(g.tour) }} · сбор в {{ meetingTime(g.tour) }}@if (g.tour.meetingPoint) {, {{ g.tour.meetingPoint }}}
            </p>
            <p class="mt-2 flex flex-wrap gap-2">
              <span class="ui-badge" [class]="phaseTone(g.phase)">{{ phaseLabel(g.phase) }}</span>
              @if (g.tour.isPublished) {
                <span class="ui-badge bg-green-100 text-green-800">в каталоге</span>
              } @else {
                <span class="ui-badge bg-red-100 text-red-800">скрыт из каталога</span>
              }
              <span class="ui-badge bg-gray-100 text-gray-700">{{ g.tour.locale }} · {{ g.tour.slug }}</span>
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            @if (g.tour.isPublished) {
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="confirmingHide.set(true)">Скрыть из каталога</button>
            } @else {
              <button type="button" class="ui-btn-primary" [disabled]="busy()" (click)="setPublished(true)">Вернуть в каталог</button>
            }
          </div>
        </div>

        @if (error()) {
          <p class="ui-error" role="alert">{{ error() }}</p>
        }
        @if (notice()) {
          <p class="text-sm text-green-700" role="status">{{ notice() }}</p>
        }
        @if (confirmingHide()) {
          <div class="ui-card flex flex-col gap-3 p-4">
            <p class="text-sm">
              Скрыть тур из каталога? Его страница перестанет открываться, новые записи прекратятся.
              Уже записанные участники останутся в группе.
            </p>
            <div class="flex gap-2">
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="setPublished(false)">Да, скрыть</button>
              <button type="button" class="ui-btn-ghost" (click)="confirmingHide.set(false)">Нет</button>
            </div>
          </div>
        }

        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div class="ui-card p-4">
            <p class="text-xs text-text-muted">Места</p>
            <p class="text-lg font-semibold text-text">{{ places(g.totals.active, g.tour.capacity) }}</p>
            <p class="text-xs text-text-muted">в листе ожидания: {{ g.totals.waitlist }}</p>
          </div>
          <div class="ui-card p-4">
            <p class="text-xs text-text-muted">Пришли на сбор</p>
            <p class="text-lg font-semibold text-text">{{ g.totals.arrived }} из {{ g.totals.active }}</p>
          </div>
          <div class="ui-card p-4">
            <p class="text-xs text-text-muted">Анкеты · правила</p>
            <p class="text-lg font-semibold text-text">{{ g.totals.forms }} · {{ g.totals.waivers }}</p>
            <p class="text-xs text-text-muted">из {{ g.totals.active }} с местом</p>
          </div>
          <div class="ui-card p-4">
            <p class="text-xs text-text-muted">Оплачено организатору</p>
            <p class="text-lg font-semibold text-text">{{ som(g.totals.paid) }}</p>
            <p class="text-xs text-text-muted">
              к оплате {{ som(g.totals.due) }}@if (g.totals.expenses) { · расходы {{ som(g.totals.expenses) }} · маржа {{ som(g.totals.margin) }}}
            </p>
          </div>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <section class="ui-card p-4">
            <h2 class="mb-3 font-semibold text-text">Тур</h2>
            <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt class="text-text-muted">Даты</dt><dd>{{ tourDates(g.tour) }}</dd>
              <dt class="text-text-muted">Сбор</dt>
              <dd>{{ formatDate(g.tour.startDate) }} в {{ meetingTime(g.tour) }}@if (g.tour.meetingPoint) {, {{ g.tour.meetingPoint }}}</dd>
              <dt class="text-text-muted">Место</dt><dd>{{ g.tour.location ?? '—' }}</dd>
              <dt class="text-text-muted">Цена</dt><dd>{{ g.tour.price === null ? '—' : som(g.tour.price) + ' с человека' }}</dd>
              <dt class="text-text-muted">Мест</dt><dd>{{ g.tour.capacity ?? 'без ограничения' }}</dd>
              <dt class="text-text-muted">Создан</dt><dd>{{ formatDateTime(g.tour.createdAt) }}</dd>
              <dt class="text-text-muted">ID</dt><dd class="break-all">{{ g.tour.id }}</dd>
            </dl>
          </section>

          <section class="ui-card p-4">
            <h2 class="mb-3 font-semibold text-text">Организатор</h2>
            <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt class="text-text-muted">Аккаунт</dt>
              <dd>
                {{ personName(g.organizer) }}
                @if (g.organizer?.username) {
                  <span class="text-text-muted">&#64;{{ g.organizer?.username }}</span>
                }
              </dd>
              @if (organizerUser(); as u) {
                @if (u.email) {
                  <dt class="text-text-muted">Email</dt><dd><a class="underline" [href]="'mailto:' + u.email">{{ u.email }}</a></dd>
                }
                @if (u.phone) {
                  <dt class="text-text-muted">Телефон</dt><dd>{{ u.phone }}</dd>
                }
              }
              <dt class="text-text-muted">Контакты тура</dt>
              <dd class="flex flex-col">
                @if (g.tour.phone) { <span>{{ g.tour.phone }}</span> }
                @if (g.tour.whatsapp) { <span>WhatsApp: {{ g.tour.whatsapp }}</span> }
                @if (g.tour.telegram) {
                  <a class="underline" [href]="telegramLink(g.tour.telegram)" target="_blank" rel="noopener">Telegram: {{ g.tour.telegram }}</a>
                }
                @if (!g.tour.phone && !g.tour.whatsapp && !g.tour.telegram) { <span>—</span> }
              </dd>
            </dl>
          </section>
        </div>

        <section class="flex flex-col gap-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="text-lg font-semibold text-text">Участники</h2>
            <div class="flex flex-wrap gap-1" role="tablist" aria-label="Участники">
              @for (f of filters; track f.value) {
                <button
                  type="button"
                  role="tab"
                  class="ui-btn-xs"
                  [class]="filter() === f.value ? 'ui-btn-primary' : 'ui-btn-soft'"
                  [attr.aria-selected]="filter() === f.value"
                  (click)="filter.set(f.value)"
                >
                  {{ f.label }} <span class="opacity-70">{{ count(f.value) }}</span>
                </button>
              }
            </div>
          </div>
          <div class="ui-table-wrap">
            <table class="ui-table">
              <thead>
                <tr><th>Участник</th><th>Статус</th><th>Записался</th><th>На сборе</th><th>Анкета</th><th>Оплата</th><th></th></tr>
              </thead>
              <tbody>
                @for (p of shown(); track p.id) {
                  <tr>
                    <td>
                      <p class="font-medium text-text">{{ participantName(p) }}</p>
                      @if (p.account?.username) {
                        <p class="text-xs text-text-muted">&#64;{{ p.account?.username }}</p>
                      }
                      @for (line of contacts(p); track line) {
                        <p class="text-xs text-text-muted">{{ line }}</p>
                      }
                      @if (!p.userId) {
                        <span class="ui-badge mt-1 bg-gray-100 text-gray-600">добавлен организатором</span>
                      }
                    </td>
                    <td>
                      <select
                        class="ui-input w-36"
                        [attr.aria-label]="'Статус: ' + participantName(p)"
                        [disabled]="busy()"
                        (change)="setStatus(p, $any($event.target).value)"
                      >
                        @for (s of statuses; track s.value) {
                          <option [value]="s.value" [selected]="s.value === p.status">{{ s.label }}</option>
                        }
                      </select>
                    </td>
                    <td class="text-text-muted">{{ formatDateTime(p.joinedAt) }}</td>
                    <td>
                      @if (hasPlace(p)) {
                        <label class="flex items-center gap-2">
                          <input
                            type="checkbox"
                            [checked]="!!p.arrivedAt"
                            [disabled]="busy()"
                            [attr.aria-label]="'Пришёл: ' + participantName(p)"
                            (change)="setArrived(p, $any($event.target).checked)"
                          />
                          @if (p.arrivedAt) {
                            <span class="text-xs text-text-muted">{{ formatDateTime(p.arrivedAt) }}</span>
                          }
                        </label>
                      } @else {
                        <span class="text-text-muted">—</span>
                      }
                    </td>
                    <td>
                      @if (p.form) {
                        <span class="ui-badge bg-green-100 text-green-800">заполнена</span>
                      } @else {
                        <span class="ui-badge bg-amber-100 text-amber-800">нет</span>
                      }
                      @if (p.waiverAt) {
                        <p class="text-xs text-text-muted">правила приняты</p>
                      } @else {
                        <p class="text-xs text-amber-700">правила не приняты</p>
                      }
                    </td>
                    <td>
                      {{ som(p.paid) }}
                      @if (p.due && hasPlace(p)) {
                        <p class="text-xs" [class]="p.paid >= p.due ? 'text-green-700' : 'text-text-muted'">из {{ som(p.due) }}</p>
                      }
                    </td>
                    <td>
                      <button type="button" class="ui-btn-ghost ui-btn-xs" [attr.aria-expanded]="open() === p.id" (click)="toggle(p.id)">
                        {{ open() === p.id ? 'Скрыть' : 'Подробнее' }}
                      </button>
                    </td>
                  </tr>
                  @if (open() === p.id) {
                    <tr>
                      <td colspan="7" class="bg-text/[0.02]">
                        <div class="grid gap-6 py-2 md:grid-cols-2">
                          <div>
                            <h3 class="mb-2 text-sm font-semibold text-text">Анкета</h3>
                            @if (p.form; as f) {
                              <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
                                <dt class="text-text-muted">Экстренный контакт</dt><dd>{{ f.emergencyName }}, {{ f.emergencyPhone }}</dd>
                                <dt class="text-text-muted">Опыт в горах</dt><dd>{{ experienceLabel(f.experience) }}</dd>
                                <dt class="text-text-muted">Здоровье</dt><dd class="whitespace-pre-line">{{ f.health || '—' }}</dd>
                                <dt class="text-text-muted">Для организатора</dt><dd class="whitespace-pre-line">{{ f.notes || '—' }}</dd>
                                @if (p.formAt) {
                                  <dt class="text-text-muted">Заполнена</dt><dd>{{ formatDateTime(p.formAt) }}</dd>
                                }
                              </dl>
                            } @else {
                              <p class="text-sm text-text-muted">Не заполнена</p>
                            }
                            @if (p.waiverAt) {
                              <p class="mt-2 text-sm text-text-muted">
                                Правила и риски ({{ p.waiverVersion }}) приняты {{ formatDateTime(p.waiverAt) }}
                              </p>
                            }
                          </div>
                          <div>
                            <h3 class="mb-2 text-sm font-semibold text-text">Оплаты организатору</h3>
                            <p class="mb-2 text-sm text-text-muted">
                              К оплате {{ som(p.due) }}@if (p.amountDue !== null) { (сумма задана организатором)}
                            </p>
                            @for (pay of p.payments; track pay.id) {
                              <p class="text-sm" [class.text-red-700]="pay.kind === 'refund'">
                                {{ formatDate(pay.paidOn) }} · {{ pay.kind === 'refund' ? 'возврат ' : '' }}{{ som(pay.amount) }} · {{ methodLabel(pay.method) }}
                                @if (pay.note) { <span class="text-text-muted">— {{ pay.note }}</span> }
                              </p>
                            } @empty {
                              <p class="text-sm text-text-muted">Оплат не отмечено</p>
                            }
                          </div>
                        </div>
                      </td>
                    </tr>
                  }
                } @empty {
                  <tr><td colspan="7" class="ui-empty">{{ g.participants.length ? 'Таких участников нет' : 'На тур пока никто не записался' }}</td></tr>
                }
              </tbody>
            </table>
          </div>
        </section>

        @if (g.staff?.length || g.expenses?.length) {
          <div class="grid gap-6 lg:grid-cols-2">
            <section class="ui-card p-4">
              <h2 class="mb-3 font-semibold text-text">Гиды и транспорт</h2>
              @for (s of g.staff ?? []; track s.id) {
                <p class="text-sm">
                  <span class="text-text-muted">{{ roleLabel(s.role) }}:</span> {{ s.name }}@if (s.phone) {, {{ s.phone }}}
                  @if (s.note) { <span class="text-text-muted">— {{ s.note }}</span> }
                </p>
              } @empty {
                <p class="text-sm text-text-muted">Никого не добавлено</p>
              }
            </section>
            <section class="ui-card p-4">
              <h2 class="mb-3 font-semibold text-text">Расходы</h2>
              @for (e of g.expenses ?? []; track e.id) {
                <p class="text-sm">
                  {{ expenseLabel(e.category) }} · {{ som(e.amount) }}
                  @if (e.note) { <span class="text-text-muted">— {{ e.note }}</span> }
                </p>
              } @empty {
                <p class="text-sm text-text-muted">Расходов не отмечено</p>
              }
            </section>
          </div>
        }
      }
    </div>

    <app-history-log class="mt-8" [modules]="['tours']" [entityId]="id()" />
  `,
})
export class TourPage {
  private readonly http = inject(HttpClient);
  // The tour's id, from the route.
  readonly id = input.required<string>();

  protected readonly group = httpResource<AdminTourGroup>(() => `/tours/admin/${this.id()}`);
  // The loaded group, replaced by what a change returns.
  protected readonly current = linkedSignal<AdminTourGroup | undefined>(() => this.group.value());
  // Accounts' contacts (the users service's admin list).
  private readonly users = httpResource<AppUser[]>(() => '/users');
  private readonly usersById = computed(
    () => new Map((this.users.value() ?? []).map((u) => [u.id, u])),
  );
  protected readonly organizerUser = computed(() => {
    const id = this.current()?.tour.userId;
    return id ? this.usersById().get(id) : undefined;
  });

  protected readonly filters = FILTERS;
  protected readonly filter = signal<Filter>('all');
  protected readonly statuses = PARTICIPANT_STATUSES;
  protected readonly shown = computed(() =>
    (this.current()?.participants ?? []).filter((p) => this.matches(p, this.filter())),
  );
  protected readonly open = signal<string | null>(null);

  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly notice = signal<string | null>(null);
  protected readonly confirmingHide = signal(false);

  protected readonly som = som;
  protected readonly personName = personName;
  protected readonly formatDate = formatDate;
  protected readonly formatDateTime = formatDateTime;
  protected readonly hasStatus = hasStatus;
  protected readonly hasPlace = hasPlace;
  protected readonly phaseLabel = phaseLabel;
  protected readonly phaseTone = phaseTone;
  protected readonly statusLabel = statusLabel;
  protected readonly places = places;
  protected readonly tourDates = tourDates;
  protected readonly meetingTime = meetingTime;
  protected readonly participantName = participantName;
  protected readonly experienceLabel = experienceLabel;
  protected readonly expenseLabel = expenseLabel;
  protected readonly roleLabel = roleLabel;
  protected readonly methodLabel = methodLabel;
  protected readonly telegramLink = telegramLink;

  protected count(filter: Filter): number {
    return (this.current()?.participants ?? []).filter((p) => this.matches(p, filter)).length;
  }

  private matches(p: TourParticipant, filter: Filter): boolean {
    switch (filter) {
      case 'place':
        return hasPlace(p);
      case 'waitlist':
      case 'cancelled':
        return p.status === filter;
      default:
        return true;
    }
  }

  // How to reach a participant: the account's contacts, else what the
  // organizer typed.
  protected contacts(p: TourParticipant): string[] {
    const user = p.userId ? this.usersById().get(p.userId) : undefined;
    const lines: string[] = [];
    if (user) {
      const name = fullName(user);
      if (name !== '—' && name !== p.account?.firstName) lines.push(name);
      if (user.phone) lines.push(user.phone);
      if (user.email) lines.push(user.email);
    }
    if (p.phone && p.phone !== user?.phone) lines.push(p.phone);
    if (p.telegram) lines.push(`Telegram: ${p.telegram}`);
    return lines;
  }

  protected toggle(id: string): void {
    this.open.update((current) => (current === id ? null : id));
  }

  protected async setStatus(p: TourParticipant, status: ParticipantStatus): Promise<void> {
    if (status === p.status) return;
    await this.change(p, { status }, `${participantName(p)}: ${statusLabel(status).toLowerCase()}`);
  }

  protected async setArrived(p: TourParticipant, arrived: boolean): Promise<void> {
    await this.change(
      p,
      { arrived },
      arrived ? `${participantName(p)} отмечен на сборе` : 'Отметка снята',
    );
  }

  private async change(
    p: TourParticipant,
    body: { status?: ParticipantStatus; arrived?: boolean },
    done: string,
  ): Promise<void> {
    await this.run(
      () =>
        firstValueFrom(
          this.http.patch<AdminTourGroup>(`/tours/admin/${this.id()}/participants/${p.id}`, body),
        ),
      done,
      'Мест нет: сначала освободите место или поставьте участника в лист ожидания',
    );
  }

  protected async setPublished(published: boolean): Promise<void> {
    await this.run(
      () =>
        firstValueFrom(
          this.http.put<AdminTourGroup>(`/tours/admin/${this.id()}/published`, { published }),
        ),
      published ? 'Тур снова в каталоге' : 'Тур скрыт из каталога',
    );
    this.confirmingHide.set(false);
  }

  private async run(
    request: () => Promise<AdminTourGroup>,
    done: string,
    conflict?: string,
  ): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    this.notice.set(null);
    try {
      this.current.set(await request());
      this.notice.set(done);
    } catch (error) {
      this.error.set(
        conflict && hasStatus(error, 409)
          ? conflict
          : apiErrorMessage(error, 'Не удалось сохранить'),
      );
      // The select shows the old status again.
      this.group.reload();
    } finally {
      this.busy.set(false);
    }
  }
}
