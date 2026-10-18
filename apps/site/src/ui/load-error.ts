import { injectResponseStatus } from '@/core/seo/response-status';
import { StateBlock } from '@/ui/state-block';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  afterNextRender,
  inject,
  output,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// What a page shows when its data didn't load; the server answers 503 so
// crawlers and caches treat it as temporary. Focus moves to the message, and
// "Retry" spins its icon while the page asks again.
@Component({
  selector: 'app-load-error',
  imports: [StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div role="alert" tabindex="-1" class="outline-none">
      <app-state-block icon="icon-[heroicons--cloud]" tone="danger" title="errors.loadFailedTitle" text="errors.loadFailedText">
        <span badge class="icon-[heroicons--x-circle] absolute right-2 bottom-2 size-[1.15rem] rounded-full bg-surface" aria-hidden="true"></span>
        <button type="button" class="ui-btn-primary group" [attr.aria-busy]="retrying()" (click)="onRetry()">
          <span class="icon-[heroicons--arrow-path] icon-anim-spin size-4" [class.animate-spin]="retrying()" aria-hidden="true"></span>{{ 'errors.retry' | transloco }}
        </button>
      </app-state-block>
    </div>
  `,
})
export class LoadError {
  readonly retry = output();
  protected readonly retrying = signal(false);

  constructor() {
    injectResponseStatus()(503);
    const host = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    afterNextRender(() =>
      host.querySelector<HTMLElement>('[role="alert"]')?.focus({ preventScroll: true }),
    );
  }

  protected onRetry(): void {
    this.retrying.set(true);
    this.retry.emit();
    // The page swaps this block out once data arrives; if it fails again, stop spinning.
    setTimeout(() => this.retrying.set(false), 1200);
  }
}
