import { CdkTrapFocus } from '@angular/cdk/a11y';
import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

let nextId = 0;

// A sheet rising from the bottom of the screen over a darkened page
// (filters, booking on phones). Escape, the backdrop or the cross close
// it; focus stays inside while open. Content is projected; a [footer]
// element sticks to the bottom.
@Component({
  selector: 'app-bottom-sheet',
  imports: [CdkTrapFocus, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (open()) {
      <div class="fixed inset-0 z-[60] bg-slate-900/42" animate.enter="motion-backdrop-in" animate.leave="motion-backdrop-out" aria-hidden="true" (click)="open.set(false)"></div>
      <section
        class="fixed inset-x-0 bottom-0 z-[61] flex max-h-[90dvh] flex-col rounded-t-[1.25rem] bg-surface text-text shadow-[0_-10px_30px_rgb(0_0_0/0.2)]"
        role="dialog"
        aria-modal="true"
        [attr.aria-labelledby]="titleId"
        cdkTrapFocus
        [cdkTrapFocusAutoCapture]="true"
        animate.enter="motion-sheet-in"
        animate.leave="motion-sheet-out"
        (keydown.escape)="open.set(false)"
      >
        <span class="mx-auto mt-2 mb-2.5 block h-[5px] w-10 shrink-0 rounded-full bg-border" aria-hidden="true"></span>
        <header class="flex items-center justify-between gap-2 px-4">
          <h2 [id]="titleId" class="flex items-center gap-2 text-base font-bold">
            <ng-content select="[icon]" />{{ title() | transloco }}
          </h2>
          <ng-content select="[action]" />
          @if (closable()) {
            <button type="button" class="ui-icon-btn -mr-1.5 size-8" [attr.aria-label]="'notify.close' | transloco" (click)="open.set(false)">
              <span class="icon-[heroicons--x-mark] size-5" aria-hidden="true"></span>
            </button>
          }
        </header>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 pt-3 pb-2">
          <ng-content />
        </div>
        <div class="px-4 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))] empty:hidden">
          <ng-content select="[footer]" />
        </div>
      </section>
    }
  `,
})
export class BottomSheet {
  readonly open = model(false);
  readonly title = input.required<string>();
  readonly closable = input(true);
  protected readonly titleId = `sheet-title-${nextId++}`;
}
