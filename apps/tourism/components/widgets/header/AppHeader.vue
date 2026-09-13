<script setup lang="ts">
import LanguageSwitcher from '~/components/features/language-switch/LanguageSwitcher.vue';
import ThemeToggle from '~/components/features/theme-toggle/ThemeToggle.vue';
import AppButton from '~/components/shared/ui/AppButton.vue';
import AppLogo from '~/components/shared/ui/AppLogo.vue';

const { isTma } = useTelegramWebApp();
const localePath = useLocalePath();
const auth = useAuth();

onMounted(() => {
  if (!auth.checked.value) {
    void auth.checkSession();
  }
});

async function handleLogout(): Promise<void> {
  await auth.logout();
}

const displayName = computed(() => auth.user.value?.firstName ?? auth.user.value?.username ?? '');
</script>

<template>
  <header
    v-if="!isTma"
    class="sticky top-0 z-40 border-b border-border bg-surface/90 backdrop-blur"
  >
    <div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
      <NuxtLink :to="localePath('/')" class="flex items-center gap-2 text-lg font-bold text-text">
        <AppLogo class="h-7 w-7 text-primary" />
        {{ $t('brand.name') }}
      </NuxtLink>
      <nav class="hidden items-center gap-6 text-sm font-medium text-text-muted sm:flex">
        <NuxtLink :to="localePath('/')" class="hover:text-text">{{ $t('nav.feed') }}</NuxtLink>
        <NuxtLink :to="localePath('/gallery')" class="hover:text-text">{{ $t('nav.gallery') }}</NuxtLink>
        <NuxtLink :to="localePath('/guides')" class="hover:text-text">{{ $t('nav.guides') }}</NuxtLink>
        <NuxtLink :to="localePath('/blog')" class="hover:text-text">{{ $t('nav.blog') }}</NuxtLink>
      </nav>
      <div class="flex items-center gap-2">
        <LanguageSwitcher />
        <ThemeToggle />
        <template v-if="auth.user.value">
          <img
            v-if="auth.user.value.avatarUrl"
            :src="auth.user.value.avatarUrl"
            :alt="displayName"
            class="h-8 w-8 rounded-full object-cover"
          />
          <span class="hidden text-sm text-text sm:inline">{{ displayName }}</span>
          <AppButton variant="ghost" @click="handleLogout">{{ $t('profile.logout') }}</AppButton>
        </template>
        <AppButton v-else variant="primary" @click="navigateTo(localePath('/login'))">
          {{ $t('login.cta') }}
        </AppButton>
      </div>
    </div>
  </header>
</template>
