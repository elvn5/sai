<script setup lang="ts">
/**
 * Переключатель светлой/тёмной темы через View Transitions API.
 * При клике новая тема «раскрывается» круговой анимацией из точки нажатия.
 * Фолбэк: нет поддержки API или включён режим «уменьшить движение» —
 * тема меняется мгновенно.
 */
const colorMode = useColorMode();
const isDark = computed(() => colorMode.value === 'dark');

function applyTheme(next: 'light' | 'dark'): Promise<void> {
  colorMode.preference = next;
  return nextTick();
}

async function toggle(event: MouseEvent): Promise<void> {
  const next: 'light' | 'dark' = isDark.value ? 'light' : 'dark';

  const prefersReducedMotion =
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // @ts-expect-error — startViewTransition ещё не во всех типах lib.dom
  if (typeof document === 'undefined' || !document.startViewTransition || prefersReducedMotion) {
    await applyTheme(next);
    return;
  }

  const x = event.clientX;
  const y = event.clientY;
  const endRadius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y),
  );

  // @ts-expect-error — startViewTransition ещё не во всех типах lib.dom
  const transition = document.startViewTransition(() => applyTheme(next));

  transition.ready.then(() => {
    // Площадь круга растёт как r², поэтому при линейном/ease-in-out радиусе
    // раскрытие ускоряется к концу и ощущается рывком. Растим радиус как √t —
    // тогда ПЛОЩАДЬ увеличивается равномерно и анимация плавная от начала до конца.
    const steps = 24;
    const clip: string[] = [];
    for (let i = 0; i <= steps; i++) {
      const p = i / steps;
      const radius = endRadius * Math.sqrt(p);
      clip.push(`circle(${radius}px at ${x}px ${y}px)`);
    }

    document.documentElement.animate(
      { clipPath: clip },
      {
        duration: 500,
        easing: 'linear',
        pseudoElement: '::view-transition-new(root)',
      },
    );
  });
}
</script>

<template>
  <ClientOnly>
    <UButton
      :icon="isDark ? 'i-lucide-moon' : 'i-lucide-sun'"
      color="neutral"
      variant="ghost"
      :aria-label="isDark ? 'Включить светлую тему' : 'Включить тёмную тему'"
      :title="isDark ? 'Светлая тема' : 'Тёмная тема'"
      @click="toggle"
    />
    <template #fallback>
      <UButton icon="i-lucide-sun" color="neutral" variant="ghost" aria-hidden="true" disabled />
    </template>
  </ClientOnly>
</template>
