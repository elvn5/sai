<script setup lang="ts">
import ColorThemeToggle from '~/components/features/color-theme-toggle/ColorThemeToggle.vue';
import LanguageSwitcher from '~/components/features/language-switch/LanguageSwitcher.vue';
import ThemeToggle from '~/components/features/theme-toggle/ThemeToggle.vue';
import AppLogo from '~/components/shared/ui/AppLogo.vue';

const { isTma, getInitData } = useTelegramWebApp();
const localePath = useLocalePath();
const route = useRoute();
const auth = useAuth();
const mobileMenuOpen = ref(false);

const navLinks = [
  { to: '/', label: 'nav.feed' },
  { to: '/trails', label: 'nav.trails' },
  { to: '/outings', label: 'nav.outings' },
  { to: '/hotels', label: 'nav.hotels' },
  { to: '/tours', label: 'nav.tours' },
  { to: '/local-guides', label: 'nav.localGuides' },
  { to: '/guides', label: 'nav.guides' },
  { to: '/gallery', label: 'nav.gallery' },
  { to: '/blog', label: 'nav.blog' },
  { to: '/hall-of-fame', label: 'nav.hallOfFame' },
] as const;

onMounted(() => {
  if (!auth.checked.value) {
    void auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
  }
});

watch(
  () => route.fullPath,
  () => {
    mobileMenuOpen.value = false;
  },
);

async function handleLogout(): Promise<void> {
  await auth.logout();
  mobileMenuOpen.value = false;
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

      <nav class="hidden items-center gap-5 text-sm font-medium text-white/75 lg:flex">
        <NuxtLink
          v-for="link in navLinks"
          :key="link.to"
          :to="localePath(link.to)"
          class="transition-colors hover:text-white"
        >
          {{ $t(link.label) }}
        </NuxtLink>
      </nav>

      <div class="hidden items-center gap-2 lg:flex">
        <div
          class="flex items-center gap-2 [&_button:not([role=menuitem])]:!text-white [&_button:hover:not([role=menuitem])]:!bg-white/10"
        >
          <LanguageSwitcher />
          <ColorThemeToggle />
          <ThemeToggle />
          <template v-if="auth.user.value">
            <NuxtLink :to="localePath('/profile')" class="flex items-center gap-2">
              <img
                v-if="auth.user.value.avatarUrl"
                :src="auth.user.value.avatarUrl"
                :alt="displayName"
                class="h-8 w-8 rounded-full object-cover ring-2 ring-white/30"
              />
              <span class="hidden text-sm text-white xl:inline">{{ displayName }}</span>
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

      <button
        type="button"
        class="flex h-10 w-10 items-center justify-center rounded-md text-white transition-colors hover:bg-white/10 lg:hidden"
        :aria-label="$t('nav.menu')"
        :aria-expanded="mobileMenuOpen"
        @click="mobileMenuOpen = !mobileMenuOpen"
      >
        <UIcon :name="mobileMenuOpen ? 'i-heroicons-x-mark' : 'i-heroicons-bars-3'" class="h-6 w-6" />
      </button>
    </div>

    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="-translate-y-2 opacity-0"
      leave-active-class="transition duration-150 ease-in"
      leave-to-class="-translate-y-2 opacity-0"
    >
      <div v-if="mobileMenuOpen" class="border-t border-white/15 bg-brand-900/95 px-4 py-4 lg:hidden">
        <nav class="flex flex-col gap-1">
          <NuxtLink
            v-for="link in navLinks"
            :key="link.to"
            :to="localePath(link.to)"
            class="rounded-md px-3 py-2.5 text-base font-medium text-white/85 transition-colors hover:bg-white/10 hover:text-white"
          >
            {{ $t(link.label) }}
          </NuxtLink>
        </nav>

        <div
          class="mt-3 flex items-center gap-2 border-t border-white/15 pt-3 [&_button:not([role=menuitem])]:!text-white [&_button:hover:not([role=menuitem])]:!bg-white/10"
        >
          <LanguageSwitcher />
          <ColorThemeToggle />
          <ThemeToggle />
        </div>

        <div class="mt-3 border-t border-white/15 pt-3">
          <template v-if="auth.user.value">
            <NuxtLink
              :to="localePath('/profile')"
              class="flex items-center gap-2 rounded-md px-3 py-2.5 text-white hover:bg-white/10"
            >
              <img
                v-if="auth.user.value.avatarUrl"
                :src="auth.user.value.avatarUrl"
                :alt="displayName"
                class="h-8 w-8 rounded-full object-cover ring-2 ring-white/30"
              />
              <span class="text-sm font-medium">{{ displayName }}</span>
            </NuxtLink>
            <UButton color="gray" variant="ghost" block class="mt-1 !text-white" @click="handleLogout">
              {{ $t('profile.logout') }}
            </UButton>
          </template>
          <UButton
            v-else
            block
            class="!bg-white !text-brand-700 hover:!bg-white/90"
            @click="navigateTo(localePath('/login'))"
          >
            {{ $t('login.cta') }}
          </UButton>
        </div>
      </div>
    </Transition>
  </header>
</template>
