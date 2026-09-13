<script setup lang="ts">
import AppLogo from '~/components/shared/ui/AppLogo.vue';

const { isTma, colorScheme } = useTelegramWebApp();

const ranks = [
  { label: 'Новичок', locked: false, color: 'gray' as const },
  { label: 'Исследователь', locked: false, color: 'secondary' as const },
  { label: 'Покоритель Ала-Арчи', locked: false, color: 'primary' as const },
  { label: '10 походных ночей', locked: true, color: 'gray' as const },
];

const nickname = ref('Новичок');
const isModalOpen = ref(false);
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-10">
    <header class="mb-10">
      <AppLogo class="h-20 w-20 text-primary-600 dark:text-primary-400" />
      <h1 class="mt-4 text-3xl font-extrabold text-text">{{ $t('brand.name') }}</h1>
      <p class="mt-2 max-w-xl text-text-muted">{{ $t('home.tagline') }}</p>
      <p class="mt-3 text-xs text-text-muted">
        TMA: <strong>{{ isTma ? 'да' : 'нет' }}</strong> · colorScheme:
        <strong>{{ colorScheme }}</strong>
      </p>
    </header>

    <article class="mb-10">
      <h2 class="mb-4 text-sm font-semibold uppercase tracking-wide text-text-muted">Buttons</h2>
      <div class="flex flex-wrap gap-3">
        <UButton color="primary">{{ $t('home.ctaPrimary') }}</UButton>
        <UButton color="secondary">{{ $t('home.ctaSecondary') }}</UButton>
        <UButton color="gray" variant="ghost" @click="isModalOpen = true">Открыть модалку</UButton>
        <UButton color="primary" disabled>Disabled</UButton>
      </div>
    </article>

    <article class="mb-10">
      <h2 class="mb-4 text-sm font-semibold uppercase tracking-wide text-text-muted">Input</h2>
      <div class="max-w-sm">
        <UFormGroup label="Никнейм">
          <UInput v-model="nickname" placeholder="Введите имя" />
        </UFormGroup>
      </div>
    </article>

    <article class="mb-10">
      <h2 class="mb-4 text-sm font-semibold uppercase tracking-wide text-text-muted">
        Badges / Achievements
      </h2>
      <div class="flex flex-wrap gap-2">
        <UBadge
          v-for="rank in ranks"
          :key="rank.label"
          :color="rank.color"
          variant="subtle"
          :class="{ 'opacity-50 grayscale': rank.locked }"
        >
          {{ rank.label }}
        </UBadge>
      </div>
    </article>

    <article class="mb-10 grid gap-4 sm:grid-cols-3">
      <UCard v-for="n in 3" :key="n">
        <USkeleton class="h-32 w-full" />
        <USkeleton class="mt-3 h-4 w-[70%]" />
        <USkeleton class="mt-2 h-3.5 w-[40%]" />
      </UCard>
    </article>

    <UModal v-model="isModalOpen">
      <div class="p-4">
        <h2 class="mb-4 text-lg font-semibold text-text">Пример модалки</h2>
        <p class="text-text-muted">Контент модального окна из дизайн-системы.</p>
        <div class="mt-4 flex justify-end">
          <UButton color="primary" @click="isModalOpen = false">Готово</UButton>
        </div>
      </div>
    </UModal>
  </section>
</template>
