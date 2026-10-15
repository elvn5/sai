import type { TelegramWidgetPayload } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { ColorModeStore } from '@/core/theme/color-mode';
import { isPlatformBrowser } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  type ElementRef,
  PLATFORM_ID,
  afterNextRender,
  effect,
  inject,
  output,
  viewChild,
} from '@angular/core';
import { TranslocoService } from '@jsverse/transloco';

const WIDGET_URL = 'https://telegram.org/js/telegram-widget.js?22';
const CALLBACK = 'onTelegramAuth';

type CallbackWindow = Window & { [CALLBACK]?: (payload: TelegramWidgetPayload) => void };

// Telegram's "Log in with Telegram" button (Login Widget). It is an iframe
// from telegram.org, created in the browser only; sign-in hands its signed
// payload to `authorized`, for POST /auth/telegram/widget.
@Component({
  selector: 'app-telegram-login-widget',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: '<div #frame class="flex min-h-14 justify-center py-2"></div>',
})
export class TelegramLoginWidget {
  readonly authorized = output<TelegramWidgetPayload>();
  private readonly frame = viewChild.required<ElementRef<HTMLElement>>('frame');
  private readonly botUsername = inject(APP_CONFIG).telegramBotUsername;
  private readonly colorMode = inject(ColorModeStore);
  private readonly transloco = inject(TranslocoService);
  private observer: MutationObserver | undefined;

  constructor() {
    if (!isPlatformBrowser(inject(PLATFORM_ID))) return;
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const view = this.frame().nativeElement.ownerDocument.defaultView as CallbackWindow;
      view[CALLBACK] = (payload) => this.authorized.emit(payload);
      destroyRef.onDestroy(() => {
        delete view[CALLBACK];
        this.observer?.disconnect();
      });
    });
    // Re-created when the mode changes: Telegram's "logged in as" chip has
    // a dark backdrop that only suits dark mode.
    effect(() => this.mount(this.colorMode.mode() === 'dark'));
  }

  private mount(userpic: boolean): void {
    const frame = this.frame().nativeElement;
    if (!this.botUsername) return;
    const script = frame.ownerDocument.createElement('script');
    script.src = WIDGET_URL;
    script.async = true;
    script.dataset['telegramLogin'] = this.botUsername;
    script.dataset['size'] = 'large';
    script.dataset['radius'] = '12';
    script.dataset['onauth'] = `${CALLBACK}(user)`;
    script.dataset['requestAccess'] = 'write';
    script.dataset['userpic'] = String(userpic);
    frame.replaceChildren(script);
    this.titleIframe(frame);
  }

  // Telegram's script adds an iframe without a title; screen readers need one.
  private titleIframe(frame: HTMLElement): void {
    this.observer?.disconnect();
    this.observer = new MutationObserver(() => {
      const iframe = frame.querySelector('iframe');
      if (!iframe) return;
      iframe.title = this.transloco.translate('login.orTelegram');
      this.observer?.disconnect();
    });
    this.observer.observe(frame, { childList: true });
  }
}
