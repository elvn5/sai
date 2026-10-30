import { apiErrorMessage } from '@/core/api/errors';
import { type AppUser, PartnerStatus } from '@/core/api/types';
import { formatDate, fullName } from '@/core/lib/format';
import { HistoryLog } from '@/ui/history-log';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  type TemplateRef,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { FormField, form, maxLength, submit } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

type PartnerKind = AppUser['partnerKind'];
type Tab = PartnerStatus.PENDING | PartnerStatus.APPROVED | PartnerStatus.REJECTED;

interface PartnerForm {
  firstName: string;
  lastName: string;
  phone: string;
  kind: PartnerKind;
  status: Tab;
}

const TABS: { status: Tab; label: string }[] = [
  { status: PartnerStatus.PENDING, label: 'Заявки' },
  { status: PartnerStatus.APPROVED, label: 'Партнёры' },
  { status: PartnerStatus.REJECTED, label: 'Отклонённые' },
];

const EMPTY_TEXT: Record<Tab, string> = {
  [PartnerStatus.PENDING]: 'Заявок нет',
  [PartnerStatus.APPROVED]: 'Партнёров пока нет',
  [PartnerStatus.REJECTED]: 'Отклонённых нет',
};

const KINDS: { value: PartnerKind; label: string }[] = [
  { value: 'organizer', label: 'Организатор туров' },
  { value: 'driver', label: 'Водитель' },
];

const STATUSES: { value: Tab; label: string }[] = [
  { value: PartnerStatus.PENDING, label: 'Заявка' },
  { value: PartnerStatus.APPROVED, label: 'Одобрен' },
  { value: PartnerStatus.REJECTED, label: 'Отклонён' },
];

// Partners and their applications (services/users): approve or reject an
// application, add an existing account as a partner, change a partner's
// kind, status and contacts, or take partnership away.
@Component({
  selector: 'app-partners-page',
  imports: [FormField, HistoryLog],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div class="flex items-center justify-between gap-4">
        <h1 class="text-xl font-semibold text-text">Партнёры</h1>
        <button type="button" class="ui-btn-primary" (click)="openCreate()">
          <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>Добавить партнёра
        </button>
      </div>

      <div class="flex gap-1 border-b border-border" role="tablist" aria-label="Статус">
        @for (t of tabs; track t.status) {
          <button
            type="button"
            role="tab"
            class="-mb-px border-b-2 px-3 py-2 text-sm"
            [class]="tab() === t.status ? 'border-brand-600 font-medium text-text' : 'border-transparent text-text-muted hover:text-text'"
            [attr.aria-selected]="tab() === t.status"
            (click)="tab.set(t.status)"
          >
            {{ t.label }}
            <span class="ml-1 rounded bg-text/5 px-1.5 text-xs">{{ counts()[t.status] }}</span>
          </button>
        }
      </div>

      <div>
        <label class="mb-3 block max-w-sm">
          <span class="sr-only">Поиск</span>
          <input
            class="ui-input"
            type="search"
            placeholder="Имя, email, телефон"
            [value]="query()"
            (input)="query.set($any($event.target).value)"
          />
        </label>
        @if (pageError()) {
          <p class="ui-error mb-3">{{ pageError() }}</p>
        }
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th>Имя</th>
                <th>Тип</th>
                <th>Контакты</th>
                <th>{{ tab() === Status.PENDING ? 'Подана' : 'Изменён' }}</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (user of visible(); track user.id) {
                <tr>
                  <td>{{ fullName(user) }}</td>
                  <td><span class="ui-badge" [class]="user.partnerKind === 'driver' ? 'bg-amber-100 text-amber-800' : 'bg-sky-100 text-sky-800'">{{ kindLabel(user.partnerKind) }}</span></td>
                  <td class="text-text-muted">{{ contact(user) }}</td>
                  <td class="text-text-muted">{{ when(user) }}</td>
                  <td class="whitespace-nowrap text-right">
                    @if (user.partnerStatus !== Status.APPROVED) {
                      <button type="button" class="ui-btn-primary ui-btn-xs" [disabled]="busyId() === user.id" (click)="decide(user, 'approve')">
                        Одобрить
                      </button>
                    }
                    @if (user.partnerStatus === Status.PENDING) {
                      <button type="button" class="ui-btn-danger ui-btn-xs" [disabled]="busyId() === user.id" (click)="decide(user, 'reject')">
                        Отклонить
                      </button>
                    }
                    <button type="button" class="ui-btn-ghost ui-btn-xs" aria-label="История" title="История" (click)="openHistory(user)">
                      <span class="icon-[heroicons--clock] size-4" aria-hidden="true"></span>
                    </button>
                    <button type="button" class="ui-btn-ghost ui-btn-xs" aria-label="Изменить" title="Изменить" (click)="openEdit(user)">
                      <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>
                    </button>
                    <button
                      type="button"
                      class="ui-btn-danger ui-btn-xs"
                      aria-label="Снять статус партнёра"
                      title="Снять статус партнёра"
                      [disabled]="busyId() === user.id"
                      (click)="remove(user)"
                    >
                      <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
                    </button>
                  </td>
                </tr>
              } @empty {
                @if (!partners.isLoading()) {
                  <tr><td colspan="5" class="ui-empty">{{ emptyText() }}</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </div>

      <app-history-log [names]="names()" [modules]="['partners']" />
    </div>

    <ng-template #editor>
      <form class="ui-dialog flex flex-col gap-4" role="dialog" aria-labelledby="partner-dialog-title" (submit)="save($event)" novalidate>
        <h2 id="partner-dialog-title" class="text-lg font-semibold text-text">Партнёр · {{ editing() ? fullName(editing()!) : '' }}</h2>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="ui-field">Имя <input class="ui-input" [formField]="partnerForm.firstName" /></label>
          <label class="ui-field">Фамилия <input class="ui-input" [formField]="partnerForm.lastName" /></label>
          <label class="ui-field">Телефон <input class="ui-input" type="tel" [formField]="partnerForm.phone" /></label>
          <label class="ui-field">
            Тип
            <select class="ui-input" [formField]="partnerForm.kind">
              @for (kind of kinds; track kind.value) {
                <option [value]="kind.value">{{ kind.label }}</option>
              }
            </select>
          </label>
          <label class="ui-field">
            Статус
            <select class="ui-input" [formField]="partnerForm.status">
              @for (status of statuses; track status.value) {
                <option [value]="status.value">{{ status.label }}</option>
              }
            </select>
          </label>
        </div>
        @if (editing(); as user) {
          <p class="ui-hint">Email: {{ user.email ?? '—' }} · Telegram: {{ user.telegramId ?? '—' }}</p>
        }
        @if (formError()) {
          <p class="ui-error">{{ formError() }}</p>
        }
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-ghost" (click)="dialogRef?.close()">Отмена</button>
          <button type="submit" class="ui-btn-primary" [disabled]="partnerForm().submitting()">Сохранить</button>
        </div>
      </form>
    </ng-template>

    <ng-template #creator>
      <form class="ui-dialog flex flex-col gap-4" role="dialog" aria-labelledby="partner-create-title" (submit)="create($event)" novalidate>
        <h2 id="partner-create-title" class="text-lg font-semibold text-text">Новый партнёр</h2>
        <p class="ui-hint">Партнёром становится существующий аккаунт: найдите пользователя и выберите тип.</p>
        <label class="ui-field">
          Пользователь
          <input
            class="ui-input"
            type="search"
            placeholder="Имя, email, телефон, @username"
            [value]="userQuery()"
            (input)="userQuery.set($any($event.target).value)"
          />
        </label>
        <ul class="max-h-64 overflow-y-auto rounded-lg border border-border" aria-label="Пользователи">
          @for (user of candidates(); track user.id) {
            <li>
              <button
                type="button"
                class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-text/5"
                [class.bg-brand-500/10]="picked()?.id === user.id"
                [attr.aria-pressed]="picked()?.id === user.id"
                (click)="picked.set(user)"
              >
                <span>{{ fullName(user) }}{{ user.username ? ' · @' + user.username : '' }}</span>
                <span class="text-xs text-text-muted">{{ contact(user) }}{{ user.partnerStatus !== Status.NONE ? ' · ' + statusLabel(user.partnerStatus) : '' }}</span>
              </button>
            </li>
          } @empty {
            <li class="ui-empty">{{ allUsers.isLoading() ? 'Загрузка…' : 'Никого не нашлось' }}</li>
          }
        </ul>
        <label class="ui-field">
          Тип
          <select class="ui-input" [value]="newKind()" (change)="newKind.set($any($event.target).value)">
            @for (kind of kinds; track kind.value) {
              <option [value]="kind.value">{{ kind.label }}</option>
            }
          </select>
        </label>
        @if (formError()) {
          <p class="ui-error">{{ formError() }}</p>
        }
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-ghost" (click)="dialogRef?.close()">Отмена</button>
          <button type="submit" class="ui-btn-primary" [disabled]="!picked() || creating()">Сделать партнёром</button>
        </div>
      </form>
    </ng-template>

    <ng-template #history>
      <div class="ui-dialog flex flex-col gap-4" role="dialog" aria-labelledby="partner-history-title">
        <h2 id="partner-history-title" class="text-lg font-semibold text-text">История · {{ historyOf() ? fullName(historyOf()!) : '' }}</h2>
        <app-history-log [modules]="['partners']" [entityId]="historyOf()?.id ?? null" [open]="true" title="Изменения партнёра" />
        <div class="flex justify-end">
          <button type="button" class="ui-btn-ghost" (click)="dialogRef?.close()">Закрыть</button>
        </div>
      </div>
    </ng-template>
  `,
})
export class PartnersPage {
  private readonly http = inject(HttpClient);
  private readonly dialog = inject(Dialog);
  private readonly editorTemplate = viewChild.required<TemplateRef<unknown>>('editor');
  private readonly creatorTemplate = viewChild.required<TemplateRef<unknown>>('creator');
  private readonly historyTemplate = viewChild.required<TemplateRef<unknown>>('history');
  private readonly historyLog = viewChild(HistoryLog);
  protected dialogRef: DialogRef | null = null;

  protected readonly Status = PartnerStatus;
  protected readonly tabs = TABS;
  protected readonly kinds = KINDS;
  protected readonly statuses = STATUSES;
  protected readonly fullName = fullName;

  protected readonly partners = httpResource<AppUser[]>(() => '/partners');
  protected readonly tab = signal<Tab>(PartnerStatus.PENDING);
  protected readonly query = signal('');
  protected readonly busyId = signal<string | null>(null);
  protected readonly pageError = signal<string | null>(null);
  protected readonly formError = signal<string | null>(null);

  protected readonly counts = computed(() => {
    const counts: Record<string, number> = Object.fromEntries(TABS.map((t) => [t.status, 0]));
    for (const user of this.partners.value() ?? []) {
      if (user.partnerStatus in counts) counts[user.partnerStatus]++;
    }
    return counts;
  });
  protected readonly visible = computed(() => {
    const tab = this.tab();
    const users = (this.partners.value() ?? []).filter(
      (u) => u.partnerStatus === tab && matches(u, this.query()),
    );
    // Applications wait in line, oldest first; the API sends newest first.
    return tab === PartnerStatus.PENDING ? users.reverse() : users;
  });
  protected readonly names = computed(() =>
    Object.fromEntries((this.partners.value() ?? []).map((u) => [u.id, fullName(u)])),
  );
  protected readonly emptyText = computed(() =>
    this.query() ? 'Никого не нашлось' : EMPTY_TEXT[this.tab()],
  );

  // Editing a partner.
  protected readonly editing = signal<AppUser | null>(null);
  protected readonly model = signal<PartnerForm>({
    firstName: '',
    lastName: '',
    phone: '',
    kind: 'organizer',
    status: PartnerStatus.APPROVED,
  });
  protected readonly partnerForm = form(this.model, (path) => {
    maxLength(path.firstName, 255);
    maxLength(path.lastName, 255);
    maxLength(path.phone, 50);
  });

  // Adding one: every account, loaded when the dialog opens.
  private readonly creatorOpen = signal(false);
  protected readonly allUsers = httpResource<AppUser[]>(() =>
    this.creatorOpen() ? '/users' : undefined,
  );
  protected readonly userQuery = signal('');
  protected readonly picked = signal<AppUser | null>(null);
  protected readonly newKind = signal<PartnerKind>('organizer');
  protected readonly creating = signal(false);
  protected readonly candidates = computed(() =>
    (this.allUsers.value() ?? [])
      .filter((u) => u.partnerStatus !== PartnerStatus.APPROVED && matches(u, this.userQuery()))
      .slice(0, 50),
  );

  protected readonly historyOf = signal<AppUser | null>(null);

  protected kindLabel(kind: PartnerKind): string {
    return KINDS.find((k) => k.value === kind)?.label ?? kind;
  }

  protected statusLabel(status: PartnerStatus): string {
    return STATUSES.find((s) => s.value === status)?.label ?? status;
  }

  protected contact(user: AppUser): string {
    return user.email ?? user.phone ?? (user.telegramId ? `tg:${user.telegramId}` : '—');
  }

  protected when(user: AppUser): string {
    const date =
      user.partnerStatus === PartnerStatus.PENDING ? user.partnerRequestedAt : user.updatedAt;
    return date ? formatDate(date) : '—';
  }

  private changed(): void {
    this.partners.reload();
    this.historyLog()?.reload();
  }

  protected async decide(user: AppUser, decision: 'approve' | 'reject'): Promise<void> {
    this.busyId.set(user.id);
    this.pageError.set(null);
    try {
      await firstValueFrom(this.http.post(`/partners/${user.id}/${decision}`, {}));
      this.changed();
    } catch (error) {
      this.pageError.set(apiErrorMessage(error, 'Не удалось сохранить решение'));
    } finally {
      this.busyId.set(null);
    }
  }

  protected async remove(user: AppUser): Promise<void> {
    const question =
      user.partnerStatus === PartnerStatus.PENDING
        ? `Удалить заявку ${fullName(user)}? Аккаунт останется, заявку можно будет подать снова.`
        : `Снять статус партнёра с ${fullName(user)}? Аккаунт останется, но без доступа к кабинету партнёра.`;
    if (!window.confirm(question)) return;
    this.busyId.set(user.id);
    this.pageError.set(null);
    try {
      await firstValueFrom(this.http.delete(`/partners/${user.id}`));
      this.changed();
    } catch (error) {
      this.pageError.set(apiErrorMessage(error, 'Не удалось снять статус'));
    } finally {
      this.busyId.set(null);
    }
  }

  protected openEdit(user: AppUser): void {
    this.editing.set(user);
    this.model.set({
      firstName: user.firstName ?? '',
      lastName: user.lastName ?? '',
      phone: user.phone ?? '',
      kind: user.partnerKind,
      status: user.partnerStatus as Tab,
    });
    this.partnerForm().reset();
    this.formError.set(null);
    this.dialogRef = this.dialog.open(this.editorTemplate());
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    const user = this.editing();
    if (!user) return;
    this.formError.set(null);
    await submit(this.partnerForm, async () => {
      const value = this.model();
      const body = {
        firstName: value.firstName.trim() || null,
        lastName: value.lastName.trim() || null,
        phone: value.phone.trim() || null,
        kind: value.kind,
        status: value.status,
      };
      try {
        await firstValueFrom(this.http.put(`/partners/${user.id}`, body));
        this.dialogRef?.close();
        this.changed();
      } catch (error) {
        this.formError.set(apiErrorMessage(error, 'Не удалось сохранить'));
      }
      return undefined;
    });
  }

  protected openCreate(): void {
    this.creatorOpen.set(true);
    this.allUsers.reload();
    this.userQuery.set('');
    this.picked.set(null);
    this.newKind.set('organizer');
    this.formError.set(null);
    this.dialogRef = this.dialog.open(this.creatorTemplate());
  }

  protected async create(event: Event): Promise<void> {
    event.preventDefault();
    const user = this.picked();
    if (!user) return;
    this.creating.set(true);
    this.formError.set(null);
    try {
      await firstValueFrom(this.http.post('/partners', { userId: user.id, kind: this.newKind() }));
      this.dialogRef?.close();
      this.tab.set(PartnerStatus.APPROVED);
      this.changed();
    } catch (error) {
      this.formError.set(apiErrorMessage(error, 'Не удалось добавить партнёра'));
    } finally {
      this.creating.set(false);
    }
  }

  protected openHistory(user: AppUser): void {
    this.historyOf.set(user);
    this.dialogRef = this.dialog.open(this.historyTemplate());
  }
}

function matches(user: AppUser, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return [
    user.firstName,
    user.lastName,
    user.username,
    user.email,
    user.phone,
    user.telegramId,
  ].some((value) => value?.toLowerCase().includes(q));
}
