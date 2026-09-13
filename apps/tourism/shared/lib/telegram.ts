import { onMounted, readonly, ref } from 'vue';

interface TelegramWebApp {
  initData: string;
  ready: () => void;
  expand: () => void;
  colorScheme: 'light' | 'dark';
  setHeaderColor: (color: string) => void;
  onEvent: (event: string, callback: () => void) => void;
  offEvent: (event: string, callback: () => void) => void;
  BackButton: {
    show: () => void;
    hide: () => void;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
  };
  HapticFeedback: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void;
    notificationOccurred: (type: 'error' | 'success' | 'warning') => void;
  };
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp };
  }
}

const isTma = ref(false);
const colorScheme = ref<'light' | 'dark'>('light');
let initialized = false;

function getWebApp(): TelegramWebApp | undefined {
  return typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined;
}

export function useTelegramWebApp() {
  onMounted(() => {
    if (initialized) return;
    initialized = true;

    const webApp = getWebApp();
    // The public telegram-web-app.js script always defines window.Telegram.WebApp,
    // even outside Telegram — a non-empty initData is the only reliable TMA signal.
    if (webApp?.initData) {
      isTma.value = true;
      colorScheme.value = webApp.colorScheme;
      webApp.ready();
      webApp.expand();
      webApp.onEvent('themeChanged', () => {
        colorScheme.value = webApp.colorScheme;
      });
    }
  });

  function setHeaderColor(color: string): void {
    getWebApp()?.setHeaderColor(color);
  }

  function showBackButton(onClick: () => void): void {
    const webApp = getWebApp();
    if (!webApp) return;
    webApp.BackButton.onClick(onClick);
    webApp.BackButton.show();
  }

  function hideBackButton(): void {
    getWebApp()?.BackButton.hide();
  }

  function haptic(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft' = 'light'): void {
    getWebApp()?.HapticFeedback.impactOccurred(style);
  }

  function getInitData(): string | undefined {
    return getWebApp()?.initData || undefined;
  }

  return {
    isTma: readonly(isTma),
    colorScheme: readonly(colorScheme),
    setHeaderColor,
    showBackButton,
    hideBackButton,
    haptic,
    getInitData,
  };
}
