import type { Achievement, LocalizedText, MyAchievements } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedText, personName } from '@/core/lib/format';
import { AchievementMedal } from '@/ui/achievements/achievement-medal';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The visitor's medals: earned ones first, then the rest of the catalog,
// greyed out with their points.
@Component({
  selector: 'app-achievement-shelf',
  imports: [AchievementMedal, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  styles: `
    .enter { animation: enter 0.6s cubic-bezier(0.22, 1, 0.36, 1) both; animation-delay: calc(var(--i) * 60ms); }
    @keyframes enter { from { opacity: 0; transform: translateY(16px); } }
    @media (prefers-reduced-motion: reduce) { .enter { animation: none; } }
  `,
  template: `
    @if (mine().awards.length > 0 || locked().length > 0) {
      <ul class="grid grid-cols-2 gap-5 sm:grid-cols-3 md:grid-cols-4">
        @for (award of mine().awards; track award.id; let i = $index) {
          <li class="enter flex flex-col items-center gap-2 text-center" [style.--i]="i">
            <app-achievement-medal [icon]="award.achievement.icon" [rarity]="award.achievement.rarity" [fresh]="award.isNew" />
            <p class="text-sm leading-tight font-semibold text-text">{{ text(award.achievement.title) }}</p>
            @if (award.awardedBy) {
              <p class="text-xs text-text-muted">{{ 'hallOfFame.awardedBy' | transloco: { name: name(award.awardedBy) } }}</p>
            }
          </li>
        }
        @for (achievement of locked(); track achievement.id; let i = $index) {
          <li class="enter flex flex-col items-center gap-2 text-center" [style.--i]="mine().awards.length + i" [title]="text(achievement.description)">
            <app-achievement-medal [icon]="achievement.icon" [rarity]="achievement.rarity" [locked]="true" />
            <p class="text-sm leading-tight font-medium text-text-muted">{{ text(achievement.title) }}</p>
            <p class="text-xs text-text-muted">+{{ achievement.points }}</p>
          </li>
        }
      </ul>
    } @else {
      <p class="py-8 text-center text-text-muted">{{ 'hallOfFame.mine.empty' | transloco }}</p>
    }
  `,
})
export class AchievementShelf {
  readonly mine = input.required<MyAchievements>();
  // The whole catalog; what isn't earned yet shows locked.
  readonly catalog = input<Achievement[]>([]);

  private readonly lang = injectLang();
  protected readonly locked = computed(() => {
    const earned = new Set(this.mine().awards.map((award) => award.achievement.id));
    return this.catalog().filter((achievement) => !earned.has(achievement.id));
  });

  protected text(value: LocalizedText): string {
    return localizedText(value, this.lang());
  }

  protected name(person: Parameters<typeof personName>[0]): string {
    return personName(person);
  }
}
