<script setup lang="ts">
import LanguageSwitcher from '~/components/features/language-switch/LanguageSwitcher.vue';
import ThemeToggle from '~/components/features/theme-toggle/ThemeToggle.vue';
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
    class="animated-gradient-border sticky top-0 z-40 bg-gradient-to-r from-brand-900 via-brand-700 to-brand-500"
  >
    <div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
      <NuxtLink :to="localePath('/')" class="flex items-center gap-2 text-lg font-bold text-white">
        <AppLogo class="h-7 w-7" />
        {{ $t('brand.name') }}
      </NuxtLink>
      <nav class="hidden items-center gap-6 text-sm font-medium text-white/75 sm:flex">
        <NuxtLink :to="localePath('/')" class="transition-colors hover:text-white">{{ $t('nav.feed') }}</NuxtLink>
        <NuxtLink :to="localePath('/gallery')" class="transition-colors hover:text-white">{{ $t('nav.gallery') }}</NuxtLink>
        <NuxtLink :to="localePath('/guides')" class="transition-colors hover:text-white">{{ $t('nav.guides') }}</NuxtLink>
        <NuxtLink :to="localePath('/blog')" class="transition-colors hover:text-white">{{ $t('nav.blog') }}</NuxtLink>
      </nav>
      <div class="flex items-center gap-2">
        <div class="flex items-center gap-2 [&_button]:!text-white [&_button:hover]:!bg-white/10">
          <LanguageSwitcher />
          <ThemeToggle />
          <template v-if="auth.user.value">
            <NuxtLink :to="localePath('/profile')" class="flex items-center gap-2">
              <img
                v-if="auth.user.value.avatarUrl"
                :src="auth.user.value.avatarUrl"
                :alt="displayName"
                class="h-8 w-8 rounded-full object-cover ring-2 ring-white/30"
              />
              <span class="hidden text-sm text-white sm:inline">{{ displayName }}</span>
            </NuxtLink>
            <UButton color="gray" variant="ghost" @click="handleLogout">
              {{ $t('profile.logout') }}
            </UButton>
          </template>
        </div>
        <UButton
          v-if="!auth.user.value"
          class="!bg-white !text-brand-700 hover:!bg-white/90"
          @click="navigateTo(localePath('/login'))"
        >
          {{ $t('login.cta') }}
        </UButton>
      </div>
    </div>
  </header>
</template>
