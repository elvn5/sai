import { injectResponseStatus } from '@/core/seo/response-status';
import { ChangeDetectionStrategy, Component, output } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// What a page shows when its data didn't load; the server answers 503 so
// crawlers and caches treat it as temporary.
@Component({
  selector: 'app-load-error',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div role="alert" class="ui-card flex flex-col items-center gap-3 px-6 py-10 text-center">
      <span class="icon-[heroicons--cloud] size-10 text-text-muted" aria-hidden="true"></span>
      <p class="text-text">{{ 'errors.loadFailed' | transloco }}</p>
      <button type="button" class="ui-btn-soft" (click)="retry.emit()">{{ 'errors.retry' | transloco }}</button>
    </div>
  `,
})
export class LoadError {
  readonly retry = output();

  constructor() {
    injectResponseStatus()(503);
  }
}
