<script setup lang="ts">
import AppLogo from '~/components/shared/ui/AppLogo.vue';

const localePath = useLocalePath();
const auth = useAuth();
const config = useRuntimeConfig();
const year = new Date().getFullYear();

const telegramUrl = computed(() => {
  const username = config.public.telegramBotUsername;
  return username ? `https://t.me/${username}` : null;
});
</script>

<template>
  <footer
    class="animated-gradient-border-top bg-gradient-to-br from-brand-900 via-brand-700 to-brand-600 text-white"
  >
    <div class="mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:grid-cols-2 lg:grid-cols-4">
      <div>
        <NuxtLink :to="localePath('/')" class="flex items-center gap-2 text-lg font-bold">
          <AppLogo class="h-7 w-7" />
          {{ $t('brand.name') }}
        </NuxtLink>
        <p class="mt-4 max-w-xs text-sm text-white/70">{{ $t('home.tagline') }}</p>
        <div class="mt-6 flex items-center gap-3">
          <a
            v-if="telegramUrl"
            :href="telegramUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 items-center justify-center rounded-full border border-white/25 transition-colors hover:border-white/60 hover:bg-white/10"
            :aria-label="$t('footer.telegram')"
          >
            <UIcon name="i-heroicons-paper-airplane" class="h-4 w-4" />
          </a>
          <NuxtLink
            :to="localePath(auth.user.value ? '/profile' : '/login')"
            class="flex h-10 w-10 items-center justify-center rounded-full border border-white/25 transition-colors hover:border-white/60 hover:bg-white/10"
            :aria-label="$t('footer.account')"
          >
            <UIcon name="i-heroicons-user" class="h-4 w-4" />
          </NuxtLink>
        </div>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wider text-white/60">
          {{ $t('footer.sections.nav') }}
        </h3>
        <ul class="mt-4 space-y-3 text-sm text-white/80">
          <li>
            <NuxtLink :to="localePath('/')" class="transition-colors hover:text-white">{{
              $t('nav.feed')
            }}</NuxtLink>
          </li>
          <li>
            <NuxtLink :to="localePath('/guides')" class="transition-colors hover:text-white">{{
              $t('nav.guides')
            }}</NuxtLink>
          </li>
          <li>
            <NuxtLink :to="localePath('/gallery')" class="transition-colors hover:text-white">{{
              $t('nav.gallery')
            }}</NuxtLink>
          </li>
          <li>
            <NuxtLink :to="localePath('/blog')" class="transition-colors hover:text-white">{{
              $t('nav.blog')
            }}</NuxtLink>
          </li>
        </ul>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wider text-white/60">
          {{ $t('footer.sections.account') }}
        </h3>
        <ul class="mt-4 space-y-3 text-sm text-white/80">
          <li v-if="auth.user.value">
            <NuxtLink :to="localePath('/profile')" class="transition-colors hover:text-white">{{
              $t('footer.profile')
            }}</NuxtLink>
          </li>
          <li v-else>
            <NuxtLink :to="localePath('/login')" class="transition-colors hover:text-white">{{
              $t('login.cta')
            }}</NuxtLink>
          </li>
          <li>
            <NuxtLink :to="localePath('/guides')" class="transition-colors hover:text-white">{{
              $t('footer.startTrip')
            }}</NuxtLink>
          </li>
        </ul>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wider text-white/60">
          {{ $t('footer.sections.contacts') }}
        </h3>
        <ul class="mt-4 space-y-3 text-sm text-white/80">
          <li class="flex items-center gap-2">
            <UIcon name="i-heroicons-map-pin" class="h-4 w-4 shrink-0 text-white/50" />
            {{ $t('footer.location') }}
          </li>
          <li v-if="telegramUrl" class="flex items-center gap-2">
            <UIcon name="i-heroicons-paper-airplane" class="h-4 w-4 shrink-0 text-white/50" />
            <a :href="telegramUrl" target="_blank" rel="noopener noreferrer" class="transition-colors hover:text-white">
              @{{ config.public.telegramBotUsername }}
            </a>
          </li>
        </ul>
      </div>
    </div>

    <div class="border-t border-white/15">
      <p class="mx-auto max-w-6xl px-4 py-5 text-center text-xs text-white/60 sm:text-left">
        {{ $t('brand.name') }} — {{ $t('footer.bottomTagline') }} © {{ year }}. {{ $t('footer.rights') }}
      </p>
    </div>
  </footer>
</template>
