import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  inject,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// How long "back online" stays before it goes.
const BACK_ONLINE_MS = 3000;

// A strip under the header while the connection is down, and a short
// "back online" when it returns. Browser only.
@Component({
  selector: 'app-network-banner',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (state(); as current) {
      <div class="ui-container pt-3" animate.enter="motion-drop-in" animate.leave="motion-drop-out">
        @if (current === 'offline') {
          <div class="ui-banner ui-banner-warning" role="status">
            <span class="icon-[heroicons--signal-slash]" aria-hidden="true"></span>
            <span><b>{{ 'notify.offline.title' | transloco }}</b> {{ 'notify.offline.text' | transloco }}</span>
            <button type="button" class="ui-btn-ghost ml-auto text-warning" (click)="reload()">{{ 'notify.offline.reload' | transloco }}</button>
          </div>
        } @else {
          <div class="ui-banner ui-banner-success" role="status">
            <span class="icon-[heroicons--signal]" aria-hidden="true"></span>
            <span>{{ 'notify.online' | transloco }}</span>
          </div>
        }
      </div>
    }
  `,
})
export class NetworkBanner {
  protected readonly state = signal<'offline' | 'online' | null>(null);

  constructor() {
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      const offline = (): void => {
        clearTimeout(timer);
        this.state.set('offline');
      };
      const online = (): void => {
        if (this.state() !== 'offline') return;
        this.state.set('online');
        timer = setTimeout(() => this.state.set(null), BACK_ONLINE_MS);
      };
      if (!navigator.onLine) offline();
      addEventListener('offline', offline);
      addEventListener('online', online);
      destroyRef.onDestroy(() => {
        clearTimeout(timer);
        removeEventListener('offline', offline);
        removeEventListener('online', online);
      });
    });
  }

  protected reload(): void {
    location.reload();
  }
}
