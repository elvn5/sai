import { apiErrorMessage, hasStatus } from '@/core/api/errors';
import {
  type Achievement,
  AchievementRarity,
  type Award,
  type LocalizedText,
} from '@/core/api/types';
import { formatDateTime, personName } from '@/core/lib/format';
import { Icon } from '@/ui/icon';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  type TemplateRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import {
  FormField,
  form,
  maxLength,
  min,
  submit,
  validateStandardSchema,
} from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';
import * as z from 'zod/mini';

type LocaleKey = 'ru' | 'ky' | 'en' | 'de';
type Texts = Record<LocaleKey, string>;

interface AchievementForm {
  title: Texts;
  description: Texts;
  icon: string;
  rarity: AchievementRarity;
  sortOrder: number;
  isActive: boolean;
}

const LOCALES: { key: LocaleKey; label: string }[] = [
  { key: 'ru', label: 'Русский' },
  { key: 'ky', label: 'Кыргызча' },
  { key: 'en', label: 'English' },
  { key: 'de', label: 'Deutsch' },
];

const RARITIES = [
  { value: AchievementRarity.COMMON, label: 'Обычное · 10 XP' },
  { value: AchievementRarity.RARE, label: 'Редкое · 25 XP' },
  { value: AchievementRarity.EPIC, label: 'Эпическое · 50 XP' },
  { value: AchievementRarity.LEGENDARY, label: 'Легендарное · 100 XP' },
];

const RARITY_BADGE: Record<AchievementRarity, string> = {
  common: 'bg-amber-100 text-amber-800',
  rare: 'bg-sky-100 text-sky-800',
  epic: 'bg-violet-100 text-violet-800',
  legendary: 'bg-yellow-200 text-yellow-900',
};

const ICON_IDEAS = [
  'i-mdi-trophy',
  'i-mdi-medal',
  'i-mdi-image-filter-hdr',
  'i-mdi-hiking',
  'i-mdi-tent',
  'i-mdi-horse',
  'i-mdi-ski',
  'i-mdi-waves',
  'i-mdi-campfire',
  'i-mdi-camera',
  'i-mdi-star-shooting',
  'i-mdi-crown',
  'i-mdi-compass',
  'i-mdi-map-marker-path',
  'i-mdi-snowflake',
  'i-mdi-heart',
];

const achievementSchema = z.object({
  title: z.object({
    ru: z.string().check(z.trim(), z.minLength(1, 'Название на русском обязательно')),
  }),
  icon: z.string().check(z.minLength(1, 'Укажите иконку')),
});

function emptyTexts(): Texts {
  return { ru: '', ky: '', en: '', de: '' };
}

// Drops empty translations: the API keeps only the languages filled in.
function compact(texts: Texts): LocalizedText {
  return Object.fromEntries(Object.entries(texts).filter(([, text]) => text.trim() !== ''));
}

// The catalog of achievements partners award on the site (services/users).
@Component({
  selector: 'app-achievements-page',
  imports: [FormField, Icon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-8">
      <section>
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h1 class="text-xl font-semibold text-text">Достижения</h1>
            <p class="text-sm text-text-muted">Каталог наград, которые партнёры вручают путешественникам на сайте.</p>
          </div>
          <button type="button" class="ui-btn-primary" (click)="openCreate()">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>Добавить
          </button>
        </div>
        @if (pageError()) {
          <p class="ui-error mb-3">{{ pageError() }}</p>
        }
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th><span class="sr-only">Иконка</span></th>
                <th>Название</th>
                <th>Редкость</th>
                <th>Порядок</th>
                <th>Статус</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (achievement of achievements.value() ?? []; track achievement.id) {
                <tr [class.opacity-50]="!achievement.isActive">
                  <td><app-icon [name]="achievement.icon" class="size-6" /></td>
                  <td>
                    <p class="font-medium">{{ achievement.title.ru }}</p>
                    @if (achievement.description.ru) {
                      <p class="text-xs text-text-muted">{{ achievement.description.ru }}</p>
                    }
                  </td>
                  <td><span class="ui-badge" [class]="rarityBadge[achievement.rarity]">{{ rarityLabel(achievement.rarity) }}</span></td>
                  <td class="text-text-muted">{{ achievement.sortOrder }}</td>
                  <td class="text-text-muted">{{ achievement.isActive ? 'Активно' : 'Скрыто' }}</td>
                  <td class="whitespace-nowrap text-right">
                    <button type="button" class="ui-btn-ghost ui-btn-xs" aria-label="Изменить" (click)="openEdit(achievement)">
                      <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>
                    </button>
                    <button type="button" class="ui-btn-danger ui-btn-xs" aria-label="Удалить" (click)="remove(achievement)">
                      <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
                    </button>
                  </td>
                </tr>
              } @empty {
                @if (!achievements.isLoading()) {
                  <tr><td colspan="6" class="ui-empty">Достижений пока нет</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-semibold text-text">Последние вручения</h2>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th>Кому</th>
                <th>Достижение</th>
                <th>Партнёр</th>
                <th>Когда</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (award of awards.value() ?? []; track award.id) {
                <tr>
                  <td>{{ person(award.user) }}</td>
                  <td>
                    <span class="inline-flex items-center gap-2">
                      <app-icon [name]="award.achievement.icon" class="size-4" />{{ award.achievement.title.ru }}
                    </span>
                    @if (award.note) {
                      <p class="text-xs italic text-text-muted">«{{ award.note }}»</p>
                    }
                  </td>
                  <td class="text-text-muted">{{ person(award.awardedBy) }}</td>
                  <td class="text-text-muted">{{ formatDateTime(award.awardedAt) }}</td>
                  <td class="text-right">
                    <button type="button" class="ui-btn-danger ui-btn-xs" (click)="revoke(award)">Отозвать</button>
                  </td>
                </tr>
              } @empty {
                @if (!awards.isLoading()) {
                  <tr><td colspan="5" class="ui-empty">Пока никто ничего не вручал</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <ng-template #editor>
      <form class="ui-dialog flex flex-col gap-4" role="dialog" aria-labelledby="achievement-dialog-title" (submit)="save($event)" novalidate>
        <h2 id="achievement-dialog-title" class="text-lg font-semibold text-text">
          {{ editingId() ? 'Редактировать достижение' : 'Новое достижение' }}
        </h2>
        <div class="grid gap-3 sm:grid-cols-2">
          @for (locale of locales; track locale.key) {
            <label class="ui-field">
              Название · {{ locale.label }}{{ locale.key === 'ru' ? ' (обязательно)' : '' }}
              <input class="ui-input" [formField]="achievementForm.title[locale.key]" />
            </label>
          }
        </div>
        @if (achievementForm.title.ru().touched() && achievementForm.title.ru().invalid()) {
          <p class="ui-error">{{ achievementForm.title.ru().errors()[0]?.message }}</p>
        }
        <div class="grid gap-3 sm:grid-cols-2">
          @for (locale of locales; track locale.key) {
            <label class="ui-field">
              Описание · {{ locale.label }}
              <textarea class="ui-input" rows="2" [formField]="achievementForm.description[locale.key]"></textarea>
            </label>
          }
        </div>

        <div class="ui-field">
          <label for="achievement-icon">Иконка</label>
          <div class="flex items-center gap-3">
            <app-icon [name]="model().icon || 'i-mdi-help'" class="size-8 shrink-0" />
            <input id="achievement-icon" class="ui-input flex-1" [formField]="achievementForm.icon" />
          </div>
          <span class="ui-hint">Имя иконки Iconify, например i-mdi-trophy</span>
          <div class="mt-1 flex flex-wrap gap-1">
            @for (icon of iconIdeas; track icon) {
              <button
                type="button"
                class="rounded border border-border p-1.5 hover:bg-text/5"
                [class.border-brand-500]="model().icon === icon"
                [class.bg-brand-500/10]="model().icon === icon"
                [title]="icon"
                [attr.aria-label]="icon"
                (click)="setIcon(icon)"
              >
                <app-icon [name]="icon" class="size-5" />
              </button>
            }
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
          <label class="ui-field">
            Редкость
            <select class="ui-input" [formField]="achievementForm.rarity">
              @for (rarity of rarities; track rarity.value) {
                <option [value]="rarity.value">{{ rarity.label }}</option>
              }
            </select>
          </label>
          <label class="ui-field">
            Порядок
            <input class="ui-input" type="number" [formField]="achievementForm.sortOrder" />
          </label>
          <label class="flex items-center gap-2 pt-6 text-sm text-text">
            <input type="checkbox" class="size-4 accent-brand-600" [formField]="achievementForm.isActive" />
            Активно
          </label>
        </div>

        @if (formError()) {
          <p class="ui-error">{{ formError() }}</p>
        }
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-ghost" (click)="dialogRef?.close()">Отмена</button>
          <button type="submit" class="ui-btn-primary" [disabled]="achievementForm().submitting()">Сохранить</button>
        </div>
      </form>
    </ng-template>
  `,
})
export class AchievementsPage {
  private readonly http = inject(HttpClient);
  private readonly dialog = inject(Dialog);
  private readonly editorTemplate = viewChild.required<TemplateRef<unknown>>('editor');
  protected dialogRef: DialogRef | null = null;

  protected readonly locales = LOCALES;
  protected readonly rarities = RARITIES;
  protected readonly rarityBadge = RARITY_BADGE;
  protected readonly iconIdeas = ICON_IDEAS;
  protected readonly formatDateTime = formatDateTime;

  protected readonly achievements = httpResource<Achievement[]>(() => '/achievements/admin/all');
  protected readonly awards = httpResource<Award[]>(() => '/achievements/admin/awards');
  protected readonly pageError = signal<string | null>(null);
  protected readonly formError = signal<string | null>(null);
  protected readonly editingId = signal<string | null>(null);

  protected readonly model = signal<AchievementForm>(this.emptyForm());
  protected readonly achievementForm = form(this.model, (path) => {
    validateStandardSchema(path, achievementSchema);
    // Limits of the API; the fields get matching maxlength/min attributes.
    for (const { key } of LOCALES) {
      maxLength(path.title[key], 80);
      maxLength(path.description[key], 300);
    }
    min(path.sortOrder, 0);
  });

  private emptyForm(): AchievementForm {
    return {
      title: emptyTexts(),
      description: emptyTexts(),
      icon: 'i-mdi-trophy',
      rarity: AchievementRarity.COMMON,
      sortOrder: 0,
      isActive: true,
    };
  }

  protected rarityLabel(rarity: AchievementRarity): string {
    return RARITIES.find((r) => r.value === rarity)?.label ?? rarity;
  }

  protected person(p: Award['user'] | null): string {
    return p ? personName(p) : '—';
  }

  protected setIcon(icon: string): void {
    this.model.update((value) => ({ ...value, icon }));
  }

  private openEditor(id: string | null, value: AchievementForm): void {
    this.editingId.set(id);
    this.model.set(value);
    this.achievementForm().reset();
    this.formError.set(null);
    this.dialogRef = this.dialog.open(this.editorTemplate());
  }

  protected openCreate(): void {
    this.openEditor(null, this.emptyForm());
  }

  protected openEdit(achievement: Achievement): void {
    this.openEditor(achievement.id, {
      title: { ...emptyTexts(), ...achievement.title },
      description: { ...emptyTexts(), ...achievement.description },
      icon: achievement.icon,
      rarity: achievement.rarity,
      sortOrder: achievement.sortOrder,
      isActive: achievement.isActive,
    });
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.formError.set(null);
    await submit(this.achievementForm, async () => {
      const value = this.model();
      const body = {
        title: compact(value.title),
        description: compact(value.description),
        icon: value.icon.trim(),
        rarity: value.rarity,
        sortOrder: Number(value.sortOrder) || 0,
        isActive: value.isActive,
      };
      const id = this.editingId();
      try {
        await firstValueFrom(
          id ? this.http.put(`/achievements/${id}`, body) : this.http.post('/achievements', body),
        );
        this.dialogRef?.close();
        this.achievements.reload();
      } catch (error) {
        this.formError.set(apiErrorMessage(error, 'Не удалось сохранить'));
      }
      return undefined;
    });
  }

  protected async remove(achievement: Achievement): Promise<void> {
    if (!window.confirm(`Удалить достижение «${achievement.title.ru}»?`)) return;
    this.pageError.set(null);
    try {
      await firstValueFrom(this.http.delete(`/achievements/${achievement.id}`));
      this.achievements.reload();
    } catch (error) {
      this.pageError.set(
        hasStatus(error, 409)
          ? `«${achievement.title.ru}» уже вручали — его можно только скрыть (снимите «Активно»).`
          : 'Не удалось удалить',
      );
    }
  }

  protected async revoke(award: Award): Promise<void> {
    const who = award.user.firstName ?? award.user.username ?? 'пользователя';
    if (!window.confirm(`Отозвать «${award.achievement.title.ru}» у ${who}?`)) return;
    await firstValueFrom(this.http.delete(`/achievements/awards/${award.id}`));
    this.awards.reload();
    this.achievements.reload();
  }
}
