import { DOCUMENT, Injectable, inject, signal } from '@angular/core';

// The part of Telegram's Mini App API (telegram-web-app.js) the site uses.
interface TelegramWebApp {
  initData: string;
  colorScheme: 'light' | 'dark';
  ready(): void;
  expand(): void;
  onEvent(event: 'themeChanged', callback: () => void): void;
  BackButton: { show(): void; hide(): void; onClick(callback: () => void): void };
  HapticFeedback: { impactOccurred(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft'): void };
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp };
  }
}

const SCRIPT_URL = 'https://telegram.org/js/telegram-web-app.js';

// Telegram opens the Mini App with its launch data in the URL fragment, and
// telegram-web-app.js keeps it in sessionStorage for later navigations.
export function isTelegramLaunch(hash: string, session: Pick<Storage, 'getItem'> | null): boolean {
  if (hash.includes('tgWebAppData')) return true;
  try {
    return Boolean(session?.getItem('__telegram__initParams'));
  } catch {
    return false;
  }
}

// The site running as a Telegram Mini App. Telegram's script loads only when
// Telegram opened the page, so ordinary visitors don't download it.
@Injectable({ providedIn: 'root' })
export class TelegramWebAppStore {
  private readonly document = inject(DOCUMENT);
  private webApp: TelegramWebApp | undefined;
  private backHandler: (() => void) | undefined;
  private readonly isTmaSignal = signal(false);
  readonly isTma = this.isTmaSignal.asReadonly();
  private readonly colorSchemeSignal = signal<'light' | 'dark'>('light');
  readonly colorScheme = this.colorSchemeSignal.asReadonly();
  // Settles once it's known whether this is a Mini App (browser only).
  readonly ready: Promise<void>;
  private resolveReady!: () => void;

  constructor() {
    this.ready = new Promise((resolve) => {
      this.resolveReady = resolve;
    });
  }

  // Browser only (afterNextRender).
  async init(): Promise<void> {
    const window = this.document.defaultView;
    if (!window || !isTelegramLaunch(window.location.hash, window.sessionStorage)) {
      this.resolveReady();
      return;
    }
    try {
      await this.loadScript();
      const webApp = window.Telegram?.WebApp;
      // The script defines WebApp outside Telegram too: initData is the signal.
      if (webApp?.initData) {
        this.webApp = webApp;
        this.isTmaSignal.set(true);
        this.colorSchemeSignal.set(webApp.colorScheme);
        webApp.onEvent('themeChanged', () => this.colorSchemeSignal.set(webApp.colorScheme));
        webApp.ready();
        webApp.expand();
      }
    } finally {
      this.resolveReady();
    }
  }

  initData(): string | undefined {
    return this.webApp?.initData || undefined;
  }

  // Telegram's own back button, shown on every page but the home page.
  setBackButton(onBack: (() => void) | null): void {
    const button = this.webApp?.BackButton;
    if (!button) return;
    if (!onBack) {
      button.hide();
      return;
    }
    if (!this.backHandler) button.onClick(() => this.backHandler?.());
    this.backHandler = onBack;
    button.show();
  }

  haptic(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft' = 'light'): void {
    this.webApp?.HapticFeedback.impactOccurred(style);
  }

  private loadScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      const script = this.document.createElement('script');
      script.src = SCRIPT_URL;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error('telegram-web-app.js did not load'));
      this.document.head.appendChild(script);
    });
  }
}
