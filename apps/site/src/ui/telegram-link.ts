import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectConfirm } from '@/ui/confirm-dialog';
import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const POLL_MS = 3000;

// Links the signed-in account to Telegram through the bot: a one-time
// deep link the visitor opens; the page waits for the bot to confirm.
// `compact` shows just the button and its status (for notices), without
// the explanation and the unlink button.
@Component({
  selector: 'app-telegram-link',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (bot) {
      <div class="flex flex-col gap-2">
        @if (linked()) {
          <div class="flex flex-wrap items-center justify-between gap-3">
            <span class="inline-flex items-center gap-2 text-sm text-text">
              <span class="icon-[mdi--telegram] size-5 text-sky-600" aria-hidden="true"></span>
              {{ (justLinked() ? 'telegramLink.justLinked' : 'telegramLink.linked') | transloco }}
              @if (auth.user()?.username; as username) {
                <span class="text-text-muted">&#64;{{ username }}</span>
              }
            </span>
            @if (!compact() && canUnlink()) {
              <button type="button" class="ui-btn-ghost ui-btn-xs" [disabled]="busy()" (click)="unlink()">{{ 'telegramLink.unlink' | transloco }}</button>
            }
          </div>
        } @else {
          @if (!compact()) {
            <p class="text-sm text-text-muted">{{ 'telegramLink.why' | transloco }}</p>
          }
          <div class="flex flex-wrap items-center gap-3">
            @if (linkUrl(); as url) {
              <a [href]="url" target="_blank" rel="noopener noreferrer" class="ui-btn-primary bg-sky-700 hover:bg-sky-800">
                <span class="icon-[mdi--telegram] size-4" aria-hidden="true"></span>{{ 'telegramLink.open' | transloco: { bot: '@' + bot } }}
              </a>
              <span class="inline-flex items-center gap-2 text-sm text-text-muted" role="status">
                <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>{{ 'telegramLink.waiting' | transloco }}
              </span>
            } @else {
              <button type="button" class="ui-btn-primary bg-sky-700 hover:bg-sky-800" [disabled]="busy()" (click)="start()">
                <span class="icon-[mdi--telegram] size-4" aria-hidden="true"></span>{{ 'telegramLink.link' | transloco }}
              </button>
            }
          </div>
        }
        @if (error(); as key) {
          <p class="ui-error" role="alert">{{ key | transloco }}</p>
        }
      </div>
    }
  `,
})
export class TelegramLink {
  readonly compact = input(false);

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  protected readonly bot = inject(APP_CONFIG).telegramBotUsername;

  protected readonly linked = computed(() => Boolean(this.auth.user()?.telegramId));
  // Unlinking would lock out an account whose only sign-in is Telegram.
  protected readonly canUnlink = computed(() => {
    const user = this.auth.user();
    return Boolean(user?.email || user?.phone);
  });
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

  protected async unlink(): Promise<void> {
    if (!(await this.confirm({ title: 'telegramLink.unlinkConfirm', danger: true }))) return;
    this.busy.set(true);
    this.error.set(null);
    try {
      await this.auth.unlinkTelegram();
      this.justLinked.set(false);
    } catch (error) {
      this.error.set(
        error instanceof HttpErrorResponse && error.status === 409
          ? 'telegramLink.onlySignIn'
          : 'telegramLink.error',
      );
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
