import type { Award } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedText, personName } from '@/core/lib/format';
import { AchievementMedal } from '@/pages/hall-of-fame/achievement-medal';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';

// The latest awards scrolling past, like a stadium ticker. The list runs
// twice for a seamless loop; the copy is hidden from screen readers.
@Component({
  selector: 'app-recent-awards-ticker',
  imports: [AchievementMedal],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .ticker { mask-image: linear-gradient(90deg, transparent, #000 8%, #000 92%, transparent); }
    .track { animation: scroll 40s linear infinite; }
    .ticker:hover .track, .ticker:focus-within .track { animation-play-state: paused; }
    @keyframes scroll { to { transform: translateX(calc(-50% - 1rem)); } }
    @media (prefers-reduced-motion: reduce) { .track { animation: none; } }
  `,
  template: `
    <div class="ticker ui-card relative overflow-hidden py-3">
      <div class="track flex w-max gap-8">
        @for (copy of [1, 2]; track copy) {
          <ul class="flex gap-8" [attr.aria-hidden]="copy === 2 ? 'true' : null">
            @for (award of awards(); track award.id) {
              <li class="flex items-center gap-3 text-sm whitespace-nowrap">
                <app-achievement-medal [icon]="award.achievement.icon" [rarity]="award.achievement.rarity" size="sm" />
                <span class="text-text">
                  <span class="font-semibold">{{ person(award) }}</span>
                  <span class="text-text-muted"> · </span>
                  {{ title(award) }}
                </span>
              </li>
            }
          </ul>
        }
      </div>
    </div>
  `,
})
export class RecentAwardsTicker {
  readonly awards = input.required<Award[]>();
  private readonly lang = injectLang();

  protected person(award: Award): string {
    return personName(award.user);
  }

  protected title(award: Award): string {
    return localizedText(award.achievement.title, this.lang());
  }
}
