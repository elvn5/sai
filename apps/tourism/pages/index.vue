<script setup lang="ts">
import AppBadge from '~/components/shared/ui/AppBadge.vue';
import AppButton from '~/components/shared/ui/AppButton.vue';
import AppCard from '~/components/shared/ui/AppCard.vue';
import AppInput from '~/components/shared/ui/AppInput.vue';
import AppLogo from '~/components/shared/ui/AppLogo.vue';
import AppModal from '~/components/shared/ui/AppModal.vue';
import SkeletonBlock from '~/components/shared/ui/SkeletonBlock.vue';

const { isTma, colorScheme } = useTelegramWebApp();

const ranks = [
  { label: 'Новичок', locked: false, tone: 'neutral' as const },
  { label: 'Исследователь', locked: false, tone: 'secondary' as const },
  { label: 'Покоритель Ала-Арчи', locked: false, tone: 'primary' as const },
  { label: '10 походных ночей', locked: true, tone: 'neutral' as const },
];

const isModalOpen = ref(false);
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-10">
    <header class="mb-10">
      <AppLogo class="h-20 w-20 text-primary" />
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
        <AppButton variant="primary">{{ $t('home.ctaPrimary') }}</AppButton>
        <AppButton variant="secondary">{{ $t('home.ctaSecondary') }}</AppButton>
        <AppButton variant="ghost" @click="isModalOpen = true">Открыть модалку</AppButton>
        <AppButton variant="primary" disabled>Disabled</AppButton>
      </div>
    </article>

    <article class="mb-10">
      <h2 class="mb-4 text-sm font-semibold uppercase tracking-wide text-text-muted">Input</h2>
      <div class="max-w-sm">
        <AppInput v-model="ranks[0]!.label" label="Никнейм" placeholder="Введите имя" />
      </div>
    </article>

    <article class="mb-10">
      <h2 class="mb-4 text-sm font-semibold uppercase tracking-wide text-text-muted">
        Badges / Achievements
      </h2>
      <div class="flex flex-wrap gap-2">
        <AppBadge
          v-for="rank in ranks"
          :key="rank.label"
          :tone="rank.tone"
          :locked="rank.locked"
        >
          {{ rank.label }}
        </AppBadge>
      </div>
    </article>

    <article class="mb-10 grid gap-4 sm:grid-cols-3">
      <AppCard v-for="n in 3" :key="n" as="article">
        <SkeletonBlock height="8rem" rounded="0.75rem" />
        <SkeletonBlock height="1rem" width="70%" rounded="0.25rem" class="mt-3" />
        <SkeletonBlock height="0.875rem" width="40%" rounded="0.25rem" class="mt-2" />
      </AppCard>
    </article>

    <AppModal v-model="isModalOpen" title="Пример модалки">
      <p class="text-text-muted">Контент модального окна из дизайн-системы.</p>
      <div class="mt-4 flex justify-end">
        <AppButton variant="primary" @click="isModalOpen = false">Готово</AppButton>
      </div>
    </AppModal>
  </section>
</template>
