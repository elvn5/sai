import { APP_CONFIG } from '@/shared/config/app-config';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Meta } from '@angular/platform-browser';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: '<router-outlet />',
})
export class App {
  constructor() {
    // Staging and previews serve the same build: keep them out of search.
    if (!inject(APP_CONFIG).indexable) {
      inject(Meta).updateTag({ name: 'robots', content: 'noindex, nofollow' });
    }
  }
}
