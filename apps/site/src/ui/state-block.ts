import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// An empty list, an error or a success: an icon in a circle (it pops in
// once), one line saying what's going on, an optional hint, and the next
// step (projected buttons). `icon` is a full Iconify class
// ("icon-[heroicons--map]") so Tailwind finds it in the caller's source.
@Component({
  selector: 'app-state-block',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="ui-state" [class.ui-card]="card()">
      <span class="ui-state-icon motion-pop" [class]="toneClass()">
        <span [class]="icon()" aria-hidden="true"></span>
        <ng-content select="[badge]" />
      </span>
      <p class="text-base font-semibold text-text">{{ title() | transloco: params() }}</p>
      @if (text()) {
        <p class="max-w-xs text-[0.8125rem] text-text-muted">{{ text() | transloco: params() }}</p>
      }
      <div class="mt-2 flex flex-wrap justify-center gap-2 empty:hidden">
        <ng-content />
      </div>
    </div>
  `,
})
export class StateBlock {
  readonly icon = input.required<string>();
  readonly title = input.required<string>();
  readonly text = input<string>();
  readonly params = input<Record<string, string | number>>({});
  readonly tone = input<'brand' | 'success' | 'danger' | 'warning'>('brand');
  // Framed as a card (lists); bare inside an existing card.
  readonly card = input(true);

  protected toneClass(): string {
    const tone = this.tone();
    return tone === 'brand' ? '' : `is-${tone}`;
  }
}
