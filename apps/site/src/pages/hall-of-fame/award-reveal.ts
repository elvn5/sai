import type { Award } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedText, personName } from '@/core/lib/format';
import { AchievementMedal } from '@/ui/achievements/achievement-medal';
import { A11yModule } from '@angular/cdk/a11y';
import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const CONFETTI_COLORS = ['#f59e0b', '#ef4444', '#22c55e', '#3b82f6', '#a855f7', '#ec4899'];
const CONFETTI = Array.from({ length: 48 }, (_, i) => ({
  left: `${(i * 37) % 100}%`,
  color: CONFETTI_COLORS[i % CONFETTI_COLORS.length],
  delay: `${(i % 12) * 0.08}s`,
  duration: `${2.2 + (i % 5) * 0.35}s`,
  drift: `${((i * 53) % 120) - 60}px`,
  spin: `${(i * 97) % 720}deg`,
}));

// Unveils newly earned achievements one by one, with confetti.
@Component({
  selector: 'app-award-reveal',
  imports: [A11yModule, AchievementMedal, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .backdrop { animation: fade 0.3s ease-out; }
    .card { animation: pop 0.6s cubic-bezier(0.34, 1.56, 0.64, 1); }
    .medal-drop { animation: drop 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) 0.15s both; }
    .rays {
      background: repeating-conic-gradient(from 0deg, color-mix(in srgb, var(--brand-500) 18%, transparent) 0deg 10deg, transparent 10deg 20deg);
      mask-image: radial-gradient(circle, #000 20%, transparent 70%);
      animation: spin 14s linear infinite;
    }
    .confetti { animation-name: fall; animation-timing-function: cubic-bezier(0.25, 0.6, 0.5, 1); animation-fill-mode: both; }
    .rarity--common { color: #a16a33; }
    .rarity--rare { color: #0284c7; }
    .rarity--epic { color: #9333ea; }
    .rarity--legendary { color: #d97706; }
    @keyframes fade { from { opacity: 0; } }
    @keyframes pop { from { opacity: 0; transform: scale(0.6) translateY(40px); } }
    @keyframes drop { from { opacity: 0; transform: translateY(-120px) rotate(-200deg) scale(0.3); } }
    @keyframes spin { to { transform: translate(-50%, -50%) rotate(360deg); } }
    @keyframes fall {
      from { transform: translate(0, -20px) rotate(0); opacity: 1; }
      to { transform: translate(var(--drift), 100vh) rotate(var(--spin)); opacity: 0.2; }
    }
    @media (prefers-reduced-motion: reduce) {
      .backdrop, .card, .medal-drop, .rays { animation: none; }
      .confetti { display: none; }
    }
  `,
  template: `
    <div
      class="backdrop fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      [attr.aria-label]="'hallOfFame.reveal.heading' | transloco"
      cdkTrapFocus
      [cdkTrapFocusAutoCapture]="true"
      (keydown.escape)="done.emit()"
    >
      @for (piece of confetti; track $index) {
        <span
          class="confetti pointer-events-none absolute top-0 h-3 w-2 rounded-sm"
          [style.left]="piece.left"
          [style.background]="piece.color"
          [style.animation-delay]="piece.delay"
          [style.animation-duration]="piece.duration"
          [style.--drift]="piece.drift"
          [style.--spin]="piece.spin"
          aria-hidden="true"
        ></span>
      }
      @if (current(); as award) {
        @for (key of [award.id]; track key) {
          <div class="card relative flex max-w-sm flex-col items-center gap-4 rounded-3xl bg-surface p-8 text-center shadow-2xl">
            <span class="rays pointer-events-none absolute top-16 left-1/2 size-72 -translate-x-1/2 -translate-y-1/2" aria-hidden="true"></span>
            <p class="relative text-xs font-bold tracking-[0.3em] text-brand-700 uppercase dark:text-brand-300">
              {{ 'hallOfFame.reveal.heading' | transloco }}
            </p>
            <div class="medal-drop relative">
              <app-achievement-medal [icon]="award.achievement.icon" [rarity]="award.achievement.rarity" size="lg" [fresh]="true" />
            </div>
            <h2 class="relative text-2xl font-extrabold text-text">{{ text(award.achievement.title) }}</h2>
            <p class="relative text-xs font-semibold tracking-widest uppercase" [class]="'rarity--' + award.achievement.rarity">
              {{ 'hallOfFame.rarity.' + award.achievement.rarity | transloco }} · +{{ award.achievement.points }}
            </p>
            @if (text(award.achievement.description); as description) {
              <p class="relative text-sm text-text-muted">{{ description }}</p>
            }
            @if (award.note) {
              <p class="relative rounded-xl bg-bg px-4 py-2 text-sm text-text italic">“{{ award.note }}”</p>
            }
            @if (award.awardedBy) {
              <p class="relative text-xs text-text-muted">
                {{ 'hallOfFame.awardedBy' | transloco: { name: person(award.awardedBy) } }}
              </p>
            }
            <button type="button" class="ui-btn-primary relative mt-2 px-5 py-2.5 text-base" cdkFocusInitial (click)="next()">
              {{ (index() < awards().length - 1 ? 'hallOfFame.reveal.next' : 'hallOfFame.reveal.done') | transloco }}
            </button>
          </div>
        }
      }
    </div>
  `,
})
export class AwardReveal {
  readonly awards = input.required<Award[]>();
  readonly done = output();
  private readonly lang = injectLang();
  protected readonly confetti = CONFETTI;
  protected readonly index = signal(0);
  protected readonly current = computed(() => this.awards()[this.index()]);

  protected next(): void {
    if (this.index() < this.awards().length - 1) this.index.update((i) => i + 1);
    else this.done.emit();
  }

  protected text(value: Award['achievement']['title']): string {
    return localizedText(value, this.lang());
  }

  protected person(value: NonNullable<Award['awardedBy']>): string {
    return personName(value);
  }
}
