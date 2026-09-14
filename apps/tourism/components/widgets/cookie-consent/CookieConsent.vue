<script setup lang="ts">
const STORAGE_KEY = 'cookie-consent';
const visible = ref(false);

onMounted(() => {
  if (!localStorage.getItem(STORAGE_KEY)) {
    visible.value = true;
  }
});

function accept(): void {
  localStorage.setItem(STORAGE_KEY, 'accepted');
  visible.value = false;
}
</script>

<template>
  <ClientOnly>
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="translate-y-full opacity-0"
      leave-active-class="transition duration-200 ease-in"
      leave-to-class="translate-y-full opacity-0"
    >
      <div
        v-if="visible"
        class="fixed inset-x-0 bottom-0 z-50 border-t border-border bg-surface px-4 py-4 shadow-lg"
        role="dialog"
        aria-live="polite"
        :aria-label="$t('cookieConsent.title')"
      >
        <div class="mx-auto flex max-w-4xl flex-col items-center gap-3 sm:flex-row sm:justify-between">
          <p class="text-sm text-text-muted">
            {{ $t('cookieConsent.message') }}
          </p>
          <UButton color="primary" class="shrink-0" @click="accept">
            {{ $t('cookieConsent.accept') }}
          </UButton>
        </div>
      </div>
    </Transition>
  </ClientOnly>
</template>
