<script setup lang="ts">
const route = useRoute();
const router = useRouter();
const { isTma, showBackButton, hideBackButton } = useTelegramWebApp();

watch(
  () => route.path,
  (path: string) => {
    if (!isTma.value) return;
    if (path === '/') {
      hideBackButton();
    } else {
      showBackButton(() => router.back());
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="min-h-screen bg-bg text-text">
    <AppHeader />
    <main>
      <slot />
    </main>
  </div>
</template>
