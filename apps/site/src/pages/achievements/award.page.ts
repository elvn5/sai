import type { Achievement, Award, AwardRecipient } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, localizedText, personName } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { AchievementMedal } from '@/ui/achievements/achievement-medal';
import { PersonAvatar } from '@/ui/person-avatar';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  afterNextRender,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { FormField, form, maxLength } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

const CODE_LENGTH = 8;

// "abcd-efgh " → "ABCDEFGH": codes are shown in two groups of four.
export function normalizeAwardCode(code: string): string {
  return code.replace(/[\s-]/g, '').toUpperCase();
}

// /achievements/award: a partner hands an achievement to a traveler by the
// code (or the QR link, ?code=) the traveler shows them. Browser only.
@Component({
  selector: 'app-award-page',
  imports: [AchievementMedal, FormField, PersonAvatar, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .enter {
      animation: enter 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
    }
    @keyframes enter {
      from {
        opacity: 0;
        transform: translateY(12px);
      }
    }
    @media (prefers-reduced-motion: reduce) {
      .enter {
        animation: none;
      }
    }
  `,
  template: `
    <section class="mx-auto max-w-4xl px-4 py-12">
      <div class="flex items-center gap-3">
        <span class="icon-[mdi--medal] size-9 text-amber-600" aria-hidden="true"></span>
        <h1 class="text-3xl font-extrabold text-text">{{ 'award.title' | transloco }}</h1>
      </div>
      <p class="mt-2 text-text-muted">{{ 'award.subtitle' | transloco }}</p>

      <form class="mt-8 flex flex-col gap-3 sm:flex-row" novalidate (submit)="lookup($event)">
        <label class="sr-only" for="award-code">{{ 'award.codePlaceholder' | transloco }}</label>
        <div class="relative flex-1">
          <span class="icon-[mdi--qrcode-scan] pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
          <input id="award-code" class="ui-input py-3 pl-10 font-mono text-lg tracking-[0.2em] uppercase" autocomplete="off" [placeholder]="'award.codePlaceholder' | transloco" [value]="code()" (input)="code.set($any($event.target).value)" />
        </div>
        <button type="submit" class="ui-btn-primary px-6 py-3 text-base" [disabled]="looking()">{{ 'award.find' | transloco }}</button>
      </form>
      @if (lookupError(); as key) {
        <p class="ui-error mt-2" role="alert">{{ key | transloco }}</p>
      }

      @if (recipient(); as person) {
        <div class="enter mt-8 flex flex-col gap-6">
          <div class="ui-card flex items-center gap-4 p-4">
            <app-person-avatar [person]="person.user" [size]="56" />
            <div>
              <p class="text-lg font-bold text-text">{{ name(person.user) }}</p>
              <p class="text-sm text-text-muted">{{ 'award.alreadyHas' | transloco: { count: person.earnedAchievementIds.length } }}</p>
            </div>
          </div>

          @if (justAwarded(); as award) {
            <div class="enter flex items-center gap-4 rounded-card border border-amber-400/50 bg-amber-400/10 p-4" role="status">
              <app-achievement-medal [icon]="award.achievement.icon" [rarity]="award.achievement.rarity" [fresh]="true" />
              <p class="font-semibold text-text">{{ 'award.success' | transloco: { name: name(award.user), title: title(award.achievement) } }}</p>
            </div>
          }

          <section aria-labelledby="choose-achievement">
            <h2 id="choose-achievement" class="mb-3 font-semibold text-text">{{ 'award.choose' | transloco }}</h2>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-3" role="group" [attr.aria-label]="'award.choose' | transloco">
              @for (achievement of catalogList(); track achievement.id) {
                <button
                  type="button"
                  class="flex flex-col items-center gap-2 rounded-card border bg-surface p-4 text-center transition disabled:cursor-not-allowed disabled:opacity-40"
                  [class]="selectedId() === achievement.id ? 'border-brand-500 ring-2 ring-brand-500' : 'border-border hover:border-brand-400'"
                  [disabled]="earned().has(achievement.id)"
                  [attr.aria-pressed]="selectedId() === achievement.id"
                  (click)="selectedId.set(achievement.id)"
                >
                  <app-achievement-medal [icon]="achievement.icon" [rarity]="achievement.rarity" />
                  <span class="text-sm font-semibold text-text">{{ title(achievement) }}</span>
                  <span class="text-xs text-text-muted">
                    @if (earned().has(achievement.id)) {
                      {{ 'award.earned' | transloco }}
                    } @else {
                      {{ 'hallOfFame.rarity.' + achievement.rarity | transloco }} · +{{ achievement.points }}
                    }
                  </span>
                </button>
              } @empty {
                @if (catalog.hasValue()) {
                  <p class="col-span-full text-sm text-text-muted">{{ 'award.emptyCatalog' | transloco }}</p>
                }
              }
            </div>
          </section>

          @if (selected(); as achievement) {
            <form class="flex flex-col gap-3" novalidate (submit)="give($event)">
              <label class="ui-field">
                {{ 'award.note' | transloco }}
                <textarea class="ui-input" rows="2" [formField]="noteForm.note"></textarea>
                <span class="ui-hint">{{ 'award.noteHint' | transloco }}</span>
              </label>
              @if (awardError(); as key) {
                <p class="ui-error" role="alert">{{ key | transloco }}</p>
              }
              <button type="submit" class="ui-btn-primary py-3 text-base" [disabled]="giving()">
                <span class="icon-[mdi--medal] size-5" aria-hidden="true"></span>{{ 'award.give' | transloco: { title: title(achievement) } }}
              </button>
            </form>
          }
        </div>
      }

      @if (givenList().length > 0) {
        <section class="mt-12" aria-labelledby="given">
          <h2 id="given" class="mb-3 font-semibold text-text">{{ 'award.history' | transloco }}</h2>
          <ul class="flex flex-col gap-2">
            @for (award of givenList(); track award.id) {
              <li class="ui-card flex items-center gap-3 px-4 py-2">
                <app-achievement-medal [icon]="award.achievement.icon" [rarity]="award.achievement.rarity" size="sm" />
                <span class="flex-1 text-sm text-text"><span class="font-semibold">{{ name(award.user) }}</span> · {{ title(award.achievement) }}</span>
                <span class="text-xs text-text-muted">{{ date(award.awardedAt) }}</span>
              </li>
            }
          </ul>
        </section>
      }
    </section>
  `,
})
export class AwardPage {
  // From the traveler's QR link.
  readonly codeParam = input<string | undefined>(undefined, { alias: 'code' });

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();

  protected readonly code = linkedSignal(() => this.codeParam() ?? '');
  protected readonly recipient = signal<AwardRecipient | null>(null);
  protected readonly looking = signal(false);
  protected readonly lookupError = signal<string | null>(null);
  protected readonly selectedId = signal<string | null>(null);
  protected readonly giving = signal(false);
  protected readonly awardError = signal<string | null>(null);
  protected readonly justAwarded = signal<Award | null>(null);

  protected readonly catalog = httpResource<Achievement[]>(() => '/achievements');
  private readonly given = httpResource<Award[]>(() => '/achievements/awards/given');
  protected readonly catalogList = computed(() => loaded(this.catalog) ?? []);
  protected readonly givenList = computed(() => loaded(this.given) ?? []);
  protected readonly earned = computed(() => new Set(this.recipient()?.earnedAchievementIds));
  protected readonly selected = computed(
    () => this.catalogList().find((item) => item.id === this.selectedId()) ?? null,
  );

  private readonly noteModel = signal({ note: '' });
  protected readonly noteForm = form(this.noteModel, (path) => {
    maxLength(path.note, 300);
  });

  constructor() {
    afterNextRender(() => {
      if (this.code()) void this.find();
    });
    bindPageMeta(() => ({
      title: this.transloco.translate('award.title'),
      description: this.transloco.translate('award.subtitle'),
      robots: 'noindex',
    }));
  }

  protected name(person: AwardRecipient['user']): string {
    return personName(person);
  }

  protected title(achievement: Achievement): string {
    return localizedText(achievement.title, this.lang());
  }

  protected date(iso: string): string {
    return formatInBishkek(this.lang(), iso, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  protected lookup(event: Event): void {
    event.preventDefault();
    void this.find();
  }

  protected async give(event: Event): Promise<void> {
    event.preventDefault();
    const achievementId = this.selectedId();
    if (!achievementId || this.noteForm().invalid()) return;
    this.giving.set(true);
    this.awardError.set(null);
    try {
      const award = await firstValueFrom(
        this.http.post<Award>('/achievements/awards', {
          code: normalizeAwardCode(this.code()),
          achievementId,
          note: this.noteModel().note.trim() || null,
        }),
      );
      this.justAwarded.set(award);
      this.recipient.update((person) =>
        person
          ? {
              ...person,
              earnedAchievementIds: [...person.earnedAchievementIds, award.achievement.id],
            }
          : person,
      );
      this.selectedId.set(null);
      this.noteModel.set({ note: '' });
      this.given.reload();
    } catch (error) {
      const status = error instanceof HttpErrorResponse ? error.status : 0;
      this.awardError.set(
        status === 409 ? 'award.duplicate' : status === 400 ? 'award.self' : 'award.awardError',
      );
    } finally {
      this.giving.set(false);
    }
  }

  private async find(): Promise<void> {
    this.lookupError.set(null);
    const code = normalizeAwardCode(this.code());
    if (code.length !== CODE_LENGTH) {
      this.lookupError.set('award.codeFormat');
      return;
    }
    this.looking.set(true);
    try {
      this.recipient.set(
        await firstValueFrom(
          this.http.get<AwardRecipient>(`/achievements/recipients/${encodeURIComponent(code)}`),
        ),
      );
      this.selectedId.set(null);
      this.justAwarded.set(null);
    } catch (error) {
      this.recipient.set(null);
      this.lookupError.set(
        error instanceof HttpErrorResponse && error.status === 404
          ? 'award.unknownCode'
          : 'award.lookupError',
      );
    } finally {
      this.looking.set(false);
    }
  }
}
