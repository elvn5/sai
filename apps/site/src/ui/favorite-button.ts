import {
  type FavoriteType,
  FavoritesStore,
  injectFavoriteToggle,
} from '@/core/favorites/favorites.store';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The heart that saves a listing. It may sit inside a card's link: the
// click goes no further. "image" is a round white button on a photo;
// "plain" sits in a toolbar.
@Component({
  selector: 'app-favorite-button',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'contents' },
  template: `
    <button
      type="button"
      class="flex shrink-0 items-center justify-center rounded-full transition-colors"
      [class]="variant() === 'image' ? 'size-9 bg-white/92 text-slate-900 shadow-sm hover:bg-white' : 'size-10 text-text-muted hover:bg-text/6 hover:text-text'"
      [attr.aria-pressed]="saved()"
      [attr.aria-label]="(saved() ? 'favorites.remove' : 'favorites.add') | transloco"
      (click)="click($event)"
    >
      @if (saved()) {
        <span class="icon-[heroicons--heart-solid] motion-pop size-5 text-red-500" aria-hidden="true"></span>
      } @else {
        <span class="icon-[heroicons--heart] size-5" aria-hidden="true"></span>
      }
    </button>
  `,
})
export class FavoriteButton {
  readonly type = input.required<FavoriteType>();
  readonly targetId = input.required<string>();
  readonly variant = input<'image' | 'plain'>('image');

  private readonly store = inject(FavoritesStore);
  private readonly toggle = injectFavoriteToggle();
  protected readonly saved = computed(() => this.store.has(this.type(), this.targetId()));

  protected click(event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    void this.toggle(this.type(), this.targetId());
  }
}
