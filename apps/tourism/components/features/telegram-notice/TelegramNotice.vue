<script setup lang="ts">
// Tells a participant whether outing notifications will reach them in
// Telegram, and how to make sure they do: notifications go to accounts
// linked to Telegram, from the site's bot, which the user must have
// started.
const auth = useAuth();
const config = useRuntimeConfig();
const botUsername = config.public.telegramBotUsername;
const linked = computed(() => Boolean(auth.user.value?.telegramId));
</script>

<template>
  <div
    v-if="auth.user.value"
    class="flex items-start gap-3 rounded-card border px-4 py-3 text-sm"
    :class="linked ? 'border-sky-500/30 bg-sky-500/10' : 'border-border bg-surface'"
  >
    <UIcon name="i-mdi-telegram" class="mt-0.5 h-5 w-5 shrink-0 text-sky-500" />
    <div class="text-text">
      <template v-if="linked">
        {{ $t('outings.telegram.linked') }}
        <a
          v-if="botUsername"
          :href="`https://t.me/${botUsername}?start=outings`"
          target="_blank"
          rel="noopener noreferrer"
          class="font-medium text-sky-600 hover:underline dark:text-sky-400"
        >
          {{ $t('outings.telegram.startBot', { bot: `@${botUsername}` }) }}
        </a>
      </template>
      <template v-else>{{ $t('outings.telegram.notLinked') }}</template>
    </div>
  </div>
</template>
