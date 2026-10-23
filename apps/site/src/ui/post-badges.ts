import type { BlogPost } from '@/core/api/types';
import {
  BLOG_DIFFICULTIES,
  BLOG_DURATION,
  BLOG_SEASONS,
  type BadgeOption,
} from '@/core/lib/blog-options';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

interface Badge extends Omit<BadgeOption, 'value'> {
  // blog.badges.* (read out before the value)
  kind: 'difficulty' | 'duration' | 'season';
  // A translation key, or the post's own text for the duration.
  label: string;
  translate: boolean;
}

// A post's difficulty, duration and season: badges whose icons play once as
// the page shows and again when the card (.group) is hovered. Renders
// nothing for a post without them.
@Component({
  selector: 'app-post-badges',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'contents' },
  template: `
    @if (badges().length > 0) {
      <ul class="flex flex-wrap gap-1.5">
        @for (badge of badges(); track badge.kind) {
          <li [class]="'ui-badge ui-badge-' + badge.tone">
            <span class="icon-intro" [style.--icon-intro]="'icon-' + badge.anim" aria-hidden="true">
              <span [class]="badge.icon + ' icon-anim-' + badge.anim"></span>
            </span>
            <span class="sr-only">{{ 'blog.badges.' + badge.kind | transloco }}:</span>
            @if (badge.translate) { {{ badge.label | transloco }} } @else { {{ badge.label }} }
          </li>
        }
      </ul>
    }
  `,
})
export class PostBadges {
  readonly post = input.required<Pick<BlogPost, 'difficulty' | 'duration' | 'season'>>();

  protected readonly badges = computed(() => {
    const { difficulty, duration, season } = this.post();
    const badges: Badge[] = [];
    const level = BLOG_DIFFICULTIES.find((option) => option.value === difficulty);
    if (level) {
      badges.push({
        ...level,
        kind: 'difficulty',
        label: `blog.difficulty.${level.value}`,
        translate: true,
      });
    }
    if (duration) {
      badges.push({ ...BLOG_DURATION, kind: 'duration', label: duration, translate: false });
    }
    const time = BLOG_SEASONS.find((option) => option.value === season);
    if (time) {
      badges.push({ ...time, kind: 'season', label: `blog.season.${time.value}`, translate: true });
    }
    return badges;
  });
}
