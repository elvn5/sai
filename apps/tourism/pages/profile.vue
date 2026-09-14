<script setup lang="ts">
const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuth();

const tabs = [
  { key: 'info', label: t('profile.tabs.info'), icon: 'i-heroicons-user-circle' },
  { key: 'history', label: t('profile.tabs.history'), icon: 'i-heroicons-map' },
];

const displayName = computed(
  () => auth.user.value?.firstName ?? auth.user.value?.username ?? auth.user.value?.email ?? '',
);

async function handleLogout(): Promise<void> {
  await auth.logout();
  await navigateTo(localePath('/'));
}

onMounted(async () => {
  if (!auth.checked.value) {
    await auth.checkSession();
  }
  if (!auth.user.value) {
    await navigateTo(localePath('/login'));
  }
});
</script>

<template>
  <section v-if="auth.user.value" class="mx-auto max-w-3xl px-4 py-12">
    <div class="flex items-center gap-4">
      <img
        v-if="auth.user.value.avatarUrl"
        :src="auth.user.value.avatarUrl"
        :alt="displayName"
        class="h-16 w-16 rounded-full object-cover"
      />
      <div
        v-else
        class="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-secondary-500 text-xl font-bold text-white"
      >
        {{ displayName.charAt(0).toUpperCase() }}
      </div>
      <div>
        <h1 class="text-2xl font-extrabold text-text">{{ displayName || $t('profile.title') }}</h1>
        <p v-if="auth.user.value.username" class="text-sm text-text-muted">@{{ auth.user.value.username }}</p>
      </div>
    </div>

    <UTabs :items="tabs" class="mt-8">
      <template #item="{ item }">
        <div v-if="item.key === 'info'" class="mt-6 flex flex-col gap-4">
          <div class="flex items-center justify-between rounded-card border border-border bg-surface px-4 py-3">
            <span class="text-sm text-text-muted">{{ $t('profile.fields.email') }}</span>
            <span class="text-sm font-medium text-text">{{ auth.user.value.email ?? '—' }}</span>
          </div>
          <div class="flex items-center justify-between rounded-card border border-border bg-surface px-4 py-3">
            <span class="text-sm text-text-muted">{{ $t('profile.fields.phone') }}</span>
            <span class="text-sm font-medium text-text">{{ auth.user.value.phone ?? '—' }}</span>
          </div>
          <div class="flex items-center justify-between rounded-card border border-border bg-surface px-4 py-3">
            <span class="text-sm text-text-muted">{{ $t('profile.fields.telegram') }}</span>
            <span class="text-sm font-medium text-text">
              {{ auth.user.value.telegramId ? '✓' : '—' }}
            </span>
          </div>

          <UButton color="gray" variant="soft" class="mt-2 self-start" @click="handleLogout">
            {{ $t('profile.logout') }}
          </UButton>
        </div>

        <div v-else-if="item.key === 'history'" class="mt-6">
          <div
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-heroicons-map" class="h-10 w-10 text-text-muted" />
            <p class="font-medium text-text">{{ $t('profile.history.empty') }}</p>
            <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.history.emptyHint') }}</p>
            <UButton color="primary" class="mt-2" @click="navigateTo(localePath('/guides'))">
              {{ $t('profile.history.cta') }}
            </UButton>
          </div>
        </div>
      </template>
    </UTabs>
  </section>
</template>
