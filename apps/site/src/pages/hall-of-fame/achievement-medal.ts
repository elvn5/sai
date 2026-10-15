import { AchievementRarity } from '@/core/api/types';
import { Icon } from '@/ui/icon';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

const SIZES = { sm: 'size-10', md: 'size-16', lg: 'size-28' } as const;
const ICON_SIZES = { sm: 'size-5', md: 'size-8', lg: 'size-14' } as const;

// A round medal for an achievement: the rarer, the brighter it glows.
// `locked` draws an unearned one as a silhouette; `fresh` adds sparkles.
@Component({
  selector: 'app-achievement-medal',
  imports: [Icon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'medal relative inline-flex shrink-0 items-center justify-center rounded-full',
    '[class]': 'hostClass()',
  },
  styles: `
    :host {
      --c1: #94a3b8;
      --c2: #475569;
      --glow: transparent;
      box-shadow: 0 0 0 0 var(--glow);
      transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }
    :host(:hover) { transform: translateY(-3px) rotate(-4deg) scale(1.06); }
    .ring { background: conic-gradient(from 0deg, var(--c1), var(--c2), var(--c1), var(--c2), var(--c1)); }
    .face {
      background: radial-gradient(circle at 30% 25%, var(--c1), var(--c2) 70%);
      box-shadow: inset 0 2px 6px rgb(255 255 255 / 0.35), inset 0 -4px 8px rgb(0 0 0 / 0.25);
    }
    :host(.medal--common) { --c1: #d6a26b; --c2: #8a5a2b; }
    :host(.medal--rare) { --c1: #7dd3fc; --c2: #0369a1; --glow: rgb(56 189 248 / 0.35); animation: glow 3s ease-in-out infinite; }
    :host(.medal--epic) { --c1: #d8b4fe; --c2: #6d28d9; --glow: rgb(168 85 247 / 0.45); animation: glow 2.4s ease-in-out infinite; }
    :host(.medal--legendary) { --c1: #fde68a; --c2: #d97706; --glow: rgb(251 191 36 / 0.6); animation: glow 1.8s ease-in-out infinite; }
    :host(.medal--legendary) .ring { animation: spin 6s linear infinite; }
    :host(.medal--locked) { --c1: rgb(148 163 184 / 0.25); --c2: rgb(71 85 105 / 0.25); filter: grayscale(1); }
    :host(.medal--locked) .face { box-shadow: inset 0 0 0 1px rgb(148 163 184 / 0.3); }
    .shine::after {
      content: '';
      position: absolute;
      inset: -50%;
      background: linear-gradient(115deg, transparent 40%, rgb(255 255 255 / 0.55) 50%, transparent 60%);
      animation: shine 3.2s ease-in-out infinite;
    }
    .spark {
      top: 50%;
      left: 50%;
      width: 6px;
      height: 6px;
      border-radius: 9999px;
      background: #fff;
      box-shadow: 0 0 8px 2px var(--c1);
      animation: spark 1.6s ease-out infinite;
      animation-delay: calc(var(--i) * 0.4s);
    }
    @keyframes glow { 50% { box-shadow: 0 0 22px 6px var(--glow); } }
    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes shine { 0%, 60% { transform: translateX(-60%); } 100% { transform: translateX(60%); } }
    @keyframes spark {
      0% { transform: translate(-50%, -50%) rotate(calc(var(--i) * 90deg)) translateY(0) scale(1); opacity: 1; }
      100% { transform: translate(-50%, -50%) rotate(calc(var(--i) * 90deg)) translateY(-160%) scale(0); opacity: 0; }
    }
    @media (prefers-reduced-motion: reduce) {
      :host, .ring, .shine::after, .spark { animation: none !important; }
    }
  `,
  template: `
    <span class="ring absolute inset-0 rounded-full"></span>
    <span class="face absolute inset-[12%] rounded-full"></span>
    @if (shiny()) {
      <span class="shine absolute inset-0 overflow-hidden rounded-full"></span>
    }
    @if (locked()) {
      <span [class]="'icon-[heroicons--lock-closed] relative text-text-muted ' + iconSize()" aria-hidden="true"></span>
    } @else {
      <app-icon [name]="icon()" [class]="'relative text-white drop-shadow ' + iconSize()" />
    }
    @if (fresh() && !locked()) {
      @for (n of sparks; track n) {
        <span class="spark absolute" [style.--i]="n"></span>
      }
    }
  `,
})
export class AchievementMedal {
  readonly icon = input.required<string>();
  readonly rarity = input.required<AchievementRarity>();
  readonly size = input<'sm' | 'md' | 'lg'>('md');
  readonly locked = input(false);
  readonly fresh = input(false);
  protected readonly sparks = [1, 2, 3, 4];

  protected readonly hostClass = computed(
    () => `${SIZES[this.size()]} ${this.locked() ? 'medal--locked' : `medal--${this.rarity()}`}`,
  );
  protected readonly iconSize = computed(() => ICON_SIZES[this.size()]);
  protected readonly shiny = computed(
    () =>
      !this.locked() &&
      (this.rarity() === AchievementRarity.EPIC || this.rarity() === AchievementRarity.LEGENDARY),
  );
}
