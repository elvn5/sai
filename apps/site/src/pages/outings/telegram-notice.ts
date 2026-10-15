import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const POLL_MS = 3000;

// Tells a participant whether outing notifications will reach them in
// Telegram: they go to accounts linked to Telegram, from the site's bot,
// which the visitor must have started. Linking is a one-time deep link to
// the bot; the page waits for the bot to confirm. (The profile page gets
// the full link/unlink control in wave 4.)
@Component({
  selector: 'app-telegram-notice',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (auth.user() && bot) {
      <div class="flex items-start gap-3 rounded-card border px-4 py-3 text-sm" [class]="linked() ? 'border-sky-500/30 bg-sky-500/10' : 'border-border bg-surface'">
        <span class="icon-[mdi--telegram] mt-0.5 size-5 shrink-0 text-sky-600" aria-hidden="true"></span>
        <div class="flex flex-col gap-2 text-text">
          @if (linked()) {
            <p>
              {{ (justLinked() ? 'telegramLink.justLinked' : 'outings.telegram.linked') | transloco }}
              <a [href]="'https://t.me/' + bot + '?start=outings'" target="_blank" rel="noopener noreferrer" class="font-medium text-sky-700 hover:underline dark:text-sky-300">
                {{ 'outings.telegram.startBot' | transloco: { bot: '@' + bot } }}
              </a>
            </p>
          } @else {
            <p>{{ 'outings.telegram.notLinked' | transloco }}</p>
            <div class="flex flex-wrap items-center gap-3">
              @if (linkUrl(); as url) {
                <a [href]="url" target="_blank" rel="noopener noreferrer" class="ui-btn-primary bg-sky-600 hover:bg-sky-700">
                  <span class="icon-[mdi--telegram] size-4" aria-hidden="true"></span>{{ 'telegramLink.open' | transloco: { bot: '@' + bot } }}
                </a>
                <span class="inline-flex items-center gap-2 text-text-muted" role="status">
                  <span class="icon-[mdi--loading] size-4 animate-spin" aria-hidden="true"></span>{{ 'telegramLink.waiting' | transloco }}
                </span>
              } @else {
                <button type="button" class="ui-btn-primary bg-sky-600 hover:bg-sky-700" [disabled]="busy()" (click)="start()">
                  <span class="icon-[mdi--telegram] size-4" aria-hidden="true"></span>{{ 'telegramLink.link' | transloco }}
                </button>
              }
            </div>
          }
          @if (error(); as key) {
            <p class="ui-error" role="alert">{{ key | transloco }}</p>
          }
        </div>
      </div>
    }
  `,
})
export class TelegramNotice {
  protected readonly auth = inject(AuthStore);
  protected readonly bot = inject(APP_CONFIG).telegramBotUsername;

  protected readonly linked = computed(() => Boolean(this.auth.user()?.telegramId));
  protected readonly linkUrl = signal<string | null>(null);
  protected readonly busy = signal(false);
  protected readonly justLinked = signal(false);
  // Translation key of the last failure.
  protected readonly error = signal<string | null>(null);
  private waitingUntil = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
  }

  protected async start(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      const { token, expiresAt } = await this.auth.startTelegramLink();
      this.linkUrl.set(`https://t.me/${this.bot}?start=link_${token}`);
      this.waitingUntil = new Date(expiresAt).getTime();
      this.poll();
    } catch {
      this.error.set('telegramLink.error');
    } finally {
      this.busy.set(false);
    }
  }

  // Re-reads the account until the bot has linked it, or the link expires.
  private poll(): void {
    clearTimeout(this.timer);
    this.timer = setTimeout(async () => {
      const user = await this.auth.refresh().catch(() => null);
      if (user?.telegramId) {
        this.linkUrl.set(null);
        this.justLinked.set(true);
      } else if (Date.now() < this.waitingUntil) {
        this.poll();
      } else {
        this.linkUrl.set(null);
        this.error.set('telegramLink.expired');
      }
    }, POLL_MS);
  }
}
