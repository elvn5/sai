<script setup lang="ts">
import type { TelegramWidgetPayload } from '~/shared/api/types';

const { t } = useI18n();
const localePath = useLocalePath();
const { isTma, getInitData } = useTelegramWebApp();
const auth = useAuth();
const config = useRuntimeConfig();

const widgetContainer = ref<HTMLDivElement | null>(null);
const loading = ref(false);
const errorMessage = ref<string | null>(null);

onMounted(async () => {
  if (isTma.value) {
    loading.value = true;
    const initData = getInitData();
    try {
      if (!initData) throw new Error('No initData');
      await auth.loginWithMiniApp(initData);
      await navigateTo(localePath('/'));
    } catch {
      errorMessage.value = t('login.error');
    } finally {
      loading.value = false;
    }
    return;
  }

  const botUsername = config.public.telegramBotUsername;
  if (!botUsername || !widgetContainer.value) return;

  (
    window as unknown as { onTelegramAuth: (payload: TelegramWidgetPayload) => void }
  ).onTelegramAuth = async (payload: TelegramWidgetPayload) => {
    try {
      await auth.loginWithWidget(payload);
      await navigateTo(localePath('/'));
    } catch {
      errorMessage.value = t('login.error');
    }
  };

  const script = document.createElement('script');
  script.src = 'https://telegram.org/js/telegram-widget.js?22';
  script.async = true;
  script.setAttribute('data-telegram-login', botUsername);
  script.setAttribute('data-size', 'large');
  script.setAttribute('data-onauth', 'onTelegramAuth(user)');
  script.setAttribute('data-request-access', 'write');
  script.setAttribute('data-radius', '12');
  widgetContainer.value.appendChild(script);
});
</script>

<template>
  <section class="mx-auto flex max-w-md flex-col items-center px-4 py-20 text-center">
    <h1 class="text-2xl font-extrabold text-text">{{ $t('login.title') }}</h1>
    <p class="mt-2 text-text-muted">{{ $t('login.subtitle') }}</p>

    <p v-if="loading" class="mt-8 text-text-muted">{{ $t('login.loading') }}</p>
    <div v-else ref="widgetContainer" class="mt-8" />

    <p v-if="errorMessage" class="mt-4 text-sm text-primary">{{ errorMessage }}</p>
  </section>
</template>
