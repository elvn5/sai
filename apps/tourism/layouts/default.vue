<script setup lang="ts">
import AppHeader from '~/components/widgets/header/AppHeader.vue';

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
