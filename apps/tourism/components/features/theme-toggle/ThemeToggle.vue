<script setup lang="ts">
const colorMode = useColorMode();
const isDark = computed(() => colorMode.value === 'dark');

function toggleTheme(event: MouseEvent): void {
  if (!document.startViewTransition) {
    colorMode.preference = isDark.value ? 'light' : 'dark';
    return;
  }

  const x = event.clientX;
  const y = event.clientY;
  const endRadius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));

  const transition = document.startViewTransition(async () => {
    colorMode.preference = isDark.value ? 'light' : 'dark';
    await nextTick();
  });

  transition.ready.then(() => {
    const clipPath = [`circle(0px at ${x}px ${y}px)`, `circle(${endRadius}px at ${x}px ${y}px)`];
    document.documentElement.animate(
      { clipPath },
      {
        duration: 500,
        easing: 'cubic-bezier(0.47, 0, 0.745, 0.715)',
        pseudoElement: '::view-transition-new(root)',
      },
    );
  });
}
</script>

<template>
  <ClientOnly>
    <UButton
      :icon="isDark ? 'i-heroicons-moon-20-solid' : 'i-heroicons-sun-20-solid'"
      color="gray"
      variant="ghost"
      :aria-label="$t('theme.toggle')"
      @click="toggleTheme"
    />
    <template #fallback>
      <span class="block h-9 w-9 rounded-full border border-border bg-surface" aria-hidden="true" />
    </template>
  </ClientOnly>
</template>
