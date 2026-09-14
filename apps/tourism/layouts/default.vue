<script setup lang="ts">
import AppFooter from '~/components/widgets/footer/AppFooter.vue';
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
  <div class="flex min-h-screen flex-col bg-bg text-text">
    <AppHeader />
    <main class="flex-1">
      <slot />
    </main>
    <AppFooter v-if="!isTma" />
  </div>
</template>
