import { ChangeDetectionStrategy, Component, RESPONSE_INIT, inject } from '@angular/core';
import { Meta } from '@angular/platform-browser';

// Unknown URLs answer with a real 404 (not a 200 "soft 404"), unindexed.
@Component({
  selector: 'app-not-found-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="mx-auto flex min-h-screen max-w-3xl flex-col items-center justify-center gap-3 px-4 text-center">
      <span class="icon-[heroicons--map] size-12 text-text-muted" aria-hidden="true"></span>
      <h1 class="text-3xl font-extrabold text-text">404</h1>
    </main>
  `,
})
export class NotFoundPage {
  constructor() {
    const response = inject(RESPONSE_INIT, { optional: true });
    if (response) response.status = 404;
    inject(Meta).updateTag({ name: 'robots', content: 'noindex' });
  }
}
