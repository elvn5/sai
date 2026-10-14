<script setup lang="ts">
import AppFooter from '~/components/widgets/footer/AppFooter.vue';
import AppHeader from '~/components/widgets/header/AppHeader.vue';
import { usePartnerReviews } from '~/shared/lib/partner-cabinet';

// The partner cabinet: the site header, then a section menu (a sidebar on
// wide screens, a scrolling row on phones and in the Telegram Mini App).
const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const localePath = useLocalePath();
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

const unansweredQuery = usePartnerReviews(ref(true), 1);
const unanswered = computed(() => unansweredQuery.data.value?.total ?? 0);

const links = computed(() => [
  {
    to: '/partner',
    label: t('cabinet.nav.dashboard'),
    icon: 'i-heroicons-squares-2x2',
    exact: true,
  },
  { to: '/partner/stats', label: t('cabinet.nav.stats'), icon: 'i-heroicons-chart-bar' },
  {
    to: '/partner/bookings',
    label: t('cabinet.nav.bookings'),
    icon: 'i-mdi-ticket-confirmation-outline',
  },
  { to: '/partner/finances', label: t('cabinet.nav.finances'), icon: 'i-heroicons-banknotes' },
  {
    to: '/partner/listings',
    label: t('cabinet.nav.listings'),
    icon: 'i-heroicons-building-storefront',
  },
  {
    to: '/partner/reviews',
    label: t('cabinet.nav.reviews'),
    icon: 'i-heroicons-chat-bubble-left-right',
    badge: unanswered.value,
  },
]);

function isActive(link: { to: string; exact?: boolean }): boolean {
  const path = localePath(link.to);
  return link.exact
    ? route.path === path
    : route.path === path || route.path.startsWith(`${path}/`);
}
</script>

<template>
  <div class="flex min-h-screen flex-col bg-bg text-text">
    <AppHeader />
    <div class="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-4 py-6 lg:flex-row lg:gap-10 lg:py-10">
      <aside class="lg:w-56 lg:shrink-0">
        <div class="lg:sticky lg:top-20">
          <p class="mb-3 hidden text-xs font-semibold uppercase tracking-wide text-text-muted lg:block">
            {{ $t('cabinet.title') }}
          </p>
          <nav class="-mx-4 flex gap-1 overflow-x-auto px-4 pb-1 lg:mx-0 lg:flex-col lg:overflow-visible lg:px-0">
            <NuxtLink
              v-for="link in links"
              :key="link.to"
              :to="localePath(link.to)"
              class="flex shrink-0 items-center gap-2 whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium transition-colors"
              :class="
                isActive(link)
                  ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300'
                  : 'text-text-muted hover:bg-surface hover:text-text'
              "
            >
              <UIcon :name="link.icon" class="h-5 w-5 shrink-0" />
              <span class="flex-1">{{ link.label }}</span>
              <span
                v-if="link.badge"
                class="rounded-full bg-brand-600 px-1.5 py-0.5 text-[11px] font-bold leading-none text-white"
              >
                {{ link.badge }}
              </span>
            </NuxtLink>
          </nav>
        </div>
      </aside>
      <main class="min-w-0 flex-1">
        <slot />
      </main>
    </div>
    <AppFooter v-if="!isTma" />
  </div>
</template>
