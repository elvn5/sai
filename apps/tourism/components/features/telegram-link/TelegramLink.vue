<script setup lang="ts">
import { ApiError } from '~/shared/api/client';

// Links the signed-in account to Telegram through the bot: a one-time
// deep link the user opens; the page waits for the bot to confirm.
// `compact` renders just the button and status, for notices.
withDefaults(defineProps<{ compact?: boolean }>(), { compact: false });

const POLL_MS = 3000;

const { t } = useI18n();
const auth = useAuth();
const config = useRuntimeConfig();
const botUsername = config.public.telegramBotUsername;

const linkUrl = ref<string | null>(null);
const waitingUntil = ref<number | null>(null);
const busy = ref(false);
const error = ref<string | null>(null);
const justLinked = ref(false);
let timer: ReturnType<typeof setTimeout> | undefined;

const linked = computed(() => Boolean(auth.user.value?.telegramId));
// Unlinking would lock out an account whose only sign-in is Telegram.
const canUnlink = computed(() => Boolean(auth.user.value?.email || auth.user.value?.phone));

async function start(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    const { token, expiresAt } = await auth.startTelegramLink();
    linkUrl.value = `https://t.me/${botUsername}?start=link_${token}`;
    waitingUntil.value = new Date(expiresAt).getTime();
    poll();
  } catch {
    error.value = t('telegramLink.error');
  } finally {
    busy.value = false;
  }
}

function poll(): void {
  clearTimeout(timer);
  timer = setTimeout(async () => {
    const user = await auth.refresh().catch(() => null);
    if (user?.telegramId) {
      linkUrl.value = null;
      waitingUntil.value = null;
      justLinked.value = true;
      return;
    }
    if (waitingUntil.value !== null && Date.now() < waitingUntil.value) {
      poll();
    } else {
      linkUrl.value = null;
      waitingUntil.value = null;
      error.value = t('telegramLink.expired');
    }
  }, POLL_MS);
}

async function unlink(): Promise<void> {
  if (!window.confirm(t('telegramLink.unlinkConfirm'))) return;
  busy.value = true;
  error.value = null;
  try {
    await auth.unlinkTelegram();
    justLinked.value = false;
  } catch (e) {
    error.value =
      e instanceof ApiError && e.status === 409
        ? t('telegramLink.onlySignIn')
        : t('telegramLink.error');
  } finally {
    busy.value = false;
  }
}

onBeforeUnmount(() => clearTimeout(timer));
</script>

<template>
  <div v-if="botUsername" class="flex flex-col gap-2">
    <div v-if="linked" class="flex flex-wrap items-center justify-between gap-3">
      <span class="inline-flex items-center gap-2 text-sm text-text">
        <UIcon name="i-mdi-telegram" class="h-5 w-5 text-sky-500" />
        {{ justLinked ? $t('telegramLink.justLinked') : $t('telegramLink.linked') }}
        <span v-if="auth.user.value?.username" class="text-text-muted">@{{ auth.user.value.username }}</span>
      </span>
      <UButton v-if="!compact && canUnlink" size="xs" color="gray" variant="ghost" :loading="busy" @click="unlink">
        {{ $t('telegramLink.unlink') }}
      </UButton>
    </div>

    <template v-else>
      <p v-if="!compact" class="text-sm text-text-muted">{{ $t('telegramLink.why') }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <UButton v-if="!linkUrl" color="primary" class="!bg-sky-500 !text-white hover:!bg-sky-600" icon="i-mdi-telegram" :loading="busy" @click="start">
          {{ $t('telegramLink.link') }}
        </UButton>
        <template v-else>
          <UButton color="primary" class="!bg-sky-500 !text-white hover:!bg-sky-600" icon="i-mdi-telegram" :to="linkUrl" target="_blank" external>
            {{ $t('telegramLink.open', { bot: `@${botUsername}` }) }}
          </UButton>
          <span class="inline-flex items-center gap-2 text-sm text-text-muted">
            <UIcon name="i-mdi-loading" class="h-4 w-4 animate-spin" />{{ $t('telegramLink.waiting') }}
          </span>
        </template>
      </div>
    </template>
    <p v-if="error" class="text-sm text-brand-600">{{ error }}</p>
  </div>
</template>
