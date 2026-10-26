import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { TelegramLink } from '@/ui/telegram-link';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Tells a participant whether tour notifications will reach them in
// Telegram: they go to accounts linked to Telegram, from the site's bot,
// which the visitor must have started.
@Component({
  selector: 'app-telegram-notice',
  imports: [TelegramLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (auth.user() && bot) {
      <div class="flex items-start gap-3 rounded-card border px-4 py-3 text-sm" [class]="linked() ? 'border-sky-500/30 bg-sky-500/10' : 'border-border bg-surface'">
        <span class="icon-[mdi--telegram] mt-0.5 size-5 shrink-0 text-sky-600" aria-hidden="true"></span>
        <div class="flex flex-col gap-2 text-text">
          @if (linked()) {
            <p>
              {{ 'tours.signup.telegram.linked' | transloco }}
              <a [href]="'https://t.me/' + bot + '?start=tours'" target="_blank" rel="noopener noreferrer" class="font-medium text-sky-700 hover:underline dark:text-sky-300">
                {{ 'tours.signup.telegram.startBot' | transloco: { bot: '@' + bot } }}
              </a>
            </p>
          } @else {
            <p>{{ 'tours.signup.telegram.notLinked' | transloco }}</p>
            <app-telegram-link [compact]="true" />
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
}
