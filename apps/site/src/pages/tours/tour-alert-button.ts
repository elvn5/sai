import { AuthStore } from '@/core/auth/auth.store';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { TelegramLink } from '@/ui/telegram-link';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  PLATFORM_ID,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// "Tell me about new tours" (to a destination, or any): the bot writes on
// Telegram when a partner publishes one. Needs Telegram linked to the
// account: without it, the link button opens here first, and the
// subscription follows once the bot confirms.
@Component({
  selector: 'app-tour-alert-button',
  imports: [TelegramLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'contents' },
  template: `
    @if (following()) {
      <button type="button" class="ui-btn-soft" [disabled]="busy()" [attr.aria-pressed]="true" (click)="unfollow()">
        <span class="icon-[heroicons--bell-alert-solid] motion-pop size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'tourAlerts.following' | transloco }}
      </button>
    } @else {
      <button type="button" class="ui-btn-soft group" [disabled]="busy()" [attr.aria-busy]="busy()" (click)="follow()">
        <span class="icon-[heroicons--bell] icon-anim-wiggle size-4" aria-hidden="true"></span>{{ 'tourAlerts.follow' | transloco }}
      </button>
    }
    @if (needsTelegram()) {
      <div class="motion-roll-in basis-full rounded-xl border border-border bg-surface p-3 text-left">
        <p class="mb-2 text-sm text-text-muted">{{ 'tourAlerts.linkFirst' | transloco }}</p>
        <app-telegram-link [compact]="true" />
      </div>
    }
  `,
})
export class TourAlertButton {
  // The destination to follow; empty for any.
  readonly destination = input<string>('');

  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthStore);
  private readonly notify = inject(Notify);
  private readonly navigate = injectSiteNavigate();
  private readonly url = injectCurrentUrl();
  private readonly followed = signal<string[]>([]);
  protected readonly busy = signal(false);
  protected readonly needsTelegram = signal(false);
  protected readonly following = computed(() => this.followed().includes(this.destination()));

  constructor() {
    if (!isPlatformBrowser(inject(PLATFORM_ID))) return;
    effect(() => {
      if (this.auth.user()) void untracked(() => this.load());
    });
    // Telegram linked while waiting here: follow as asked.
    effect(() => {
      if (this.needsTelegram() && this.auth.user()?.telegramId)
        untracked(() => {
          this.needsTelegram.set(false);
          void this.follow();
        });
    });
  }

  protected async follow(): Promise<void> {
    if (!(await this.auth.ensureSession())) {
      await this.navigate(`/login?redirect=${encodeURIComponent(this.url())}`);
      return;
    }
    if (!this.auth.user()?.telegramId) {
      this.needsTelegram.set(true);
      return;
    }
    await this.send('put', 'tourAlerts.followed');
  }

  protected async unfollow(): Promise<void> {
    await this.send('delete', 'tourAlerts.unfollowed');
  }

  private async send(method: 'put' | 'delete', done: string): Promise<void> {
    this.busy.set(true);
    try {
      const res = await firstValueFrom(
        this.http.request<{ destinations: string[] }>(method.toUpperCase(), '/tour-alerts', {
          body: { destination: this.destination() },
        }),
      );
      this.followed.set(res.destinations);
      this.notify.success({ title: done });
    } catch (error) {
      const status = error instanceof HttpErrorResponse ? error.status : 0;
      if (status === 409) this.needsTelegram.set(true);
      else
        this.notify.error({
          title: status === 503 ? 'tourAlerts.unavailable' : 'tourAlerts.failed',
          text: 'notify.tryAgain',
        });
    } finally {
      this.busy.set(false);
    }
  }

  private async load(): Promise<void> {
    try {
      const res = await firstValueFrom(
        this.http.get<{ destinations: string[] }>('/tour-alerts/mine'),
      );
      this.followed.set(res.destinations);
    } catch {
      // Shown as not following; following again is harmless.
    }
  }
}
