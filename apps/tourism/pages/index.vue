<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import HotelCard from '~/components/entities/hotel/HotelCard.vue';
import OutingCard from '~/components/entities/outing/OutingCard.vue';
import AppLogo from '~/components/shared/ui/AppLogo.vue';
import { useApiClient } from '~/shared/api/client';
import {
  type BlogPost,
  type Hotel,
  type Outing,
  type Paginated,
  SeoPage,
} from '~/shared/api/types';
import { FEATURED_DESTINATIONS } from '~/shared/lib/listing-options';

const NEWS_LIMIT = 3;
const TOP_HOTELS_LIMIT = 3;

const QUICK_LINKS = [
  { to: '/hotels', icon: 'i-heroicons-building-office-2', key: 'hotels' },
  { to: '/tours', icon: 'i-heroicons-map', key: 'tours' },
  { to: '/local-guides', icon: 'i-heroicons-user-group', key: 'localGuides' },
] as const;

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const postsQuery = useQuery({
  queryKey: ['blog', locale],
  queryFn: () => api.get<BlogPost[]>(`/blog?locale=${locale.value}`),
});
await postsQuery.suspense();

const latestPosts = computed(() => (postsQuery.data.value ?? []).slice(0, NEWS_LIMIT));

const topHotelsQuery = useQuery({
  queryKey: ['hotels', 'top'],
  queryFn: () => api.get<Paginated<Hotel>>(`/hotels?sort=rating&limit=${TOP_HOTELS_LIMIT}`),
});
await topHotelsQuery.suspense();
const topHotels = computed(() => topHotelsQuery.data.value?.items ?? []);

// Loaded in the browser: it changes too often to matter for SEO.
const outingsQuery = useQuery({
  queryKey: ['outings', 'home'],
  queryFn: () => api.get<Outing[]>('/outings?limit=4'),
  enabled: import.meta.client,
});
const upcomingOutings = computed(() => outingsQuery.data.value ?? []);

const pageSeoQuery = usePageSeo(SeoPage.HOME, locale);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || undefined,
  description: () => pageSeo.value?.description || t('home.tagline'),
  ogTitle: () => pageSeo.value?.title || t('brand.name'),
  ogDescription: () => pageSeo.value?.description || t('home.tagline'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl
      ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}`
      : '/images/hero-mountain.webp',
});
</script>

<template>
  <div>
    <section class="relative isolate overflow-hidden px-4 py-24 text-center sm:py-32">
      <div
        class="absolute inset-0 -z-20 bg-cover bg-center bg-no-repeat"
        style="background-image: url('/images/hero-mountain.webp')"
      />
      <div class="absolute inset-0 -z-10 bg-gradient-to-b from-black/70 via-black/50 to-bg" />

      <div class="mx-auto max-w-4xl">
        <AppLogo class="mx-auto h-16 w-16 text-white" />
        <h1 class="mt-4 text-3xl font-extrabold text-white sm:text-4xl">{{ $t('brand.name') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-white/85">{{ $t('home.tagline') }}</p>
        <div class="mt-6 flex flex-wrap justify-center gap-3">
          <UButton color="primary" size="lg" :to="localePath('/hotels')">
            {{ $t('home.ctaHotels') }}
          </UButton>
          <UButton size="lg" class="!bg-white/15 !text-white hover:!bg-white/25" :to="localePath('/guides')">
            {{ $t('home.ctaPrimary') }}
          </UButton>
        </div>
      </div>
    </section>

    <section class="relative z-10 mx-auto -mt-10 max-w-4xl px-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <NuxtLink
          v-for="link in QUICK_LINKS"
          :key="link.key"
          :to="localePath(link.to)"
          class="flex items-center gap-3 rounded-card border border-border bg-surface p-4 shadow-sm transition hover:shadow-md"
        >
          <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-brand-500/10">
            <UIcon :name="link.icon" class="h-6 w-6 text-brand-600 dark:text-brand-400" />
          </span>
          <span>
            <span class="block font-semibold text-text">{{ $t(`home.quick.${link.key}.title`) }}</span>
            <span class="block text-sm text-text-muted">{{ $t(`home.quick.${link.key}.description`) }}</span>
          </span>
        </NuxtLink>
      </div>
    </section>

    <section v-if="upcomingOutings.length > 0" class="mx-auto max-w-4xl px-4 pt-14">
      <div class="mb-4 flex items-center justify-between">
        <div>
          <h2 class="text-xl font-bold text-text">{{ $t('home.outingsHeading') }}</h2>
          <p class="mt-1 text-sm text-text-muted">{{ $t('home.outingsSubheading') }}</p>
        </div>
        <NuxtLink :to="localePath('/outings')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
          {{ $t('home.viewAllOutings') }}
        </NuxtLink>
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <OutingCard v-for="outing in upcomingOutings" :key="outing.id" :outing="outing" />
      </div>
    </section>

    <section class="mx-auto max-w-4xl px-4 py-14">
      <h2 class="text-xl font-bold text-text">{{ $t('home.destinationsHeading') }}</h2>
      <p class="mt-1 text-sm text-text-muted">{{ $t('home.destinationsSubheading') }}</p>
      <div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="destination in FEATURED_DESTINATIONS"
          :key="destination"
          class="flex flex-col rounded-card border border-border bg-gradient-to-br from-brand-500/10 via-surface to-secondary-500/10 p-5"
        >
          <h3 class="flex items-center gap-2 text-lg font-semibold text-text">
            <UIcon name="i-heroicons-map-pin" class="h-5 w-5 text-brand-600" />
            {{ $t(`destinations.${destination}`) }}
          </h3>
          <p class="mt-2 flex-1 text-sm text-text-muted">{{ $t(`home.destinationBlurbs.${destination}`) }}</p>
          <div class="mt-4 flex flex-wrap gap-x-4 gap-y-1 text-sm font-medium">
            <NuxtLink :to="localePath(`/hotels?destination=${destination}`)" class="text-primary-600 hover:underline dark:text-primary-400">
              {{ $t('nav.hotels') }}
            </NuxtLink>
            <NuxtLink :to="localePath(`/tours?destination=${destination}`)" class="text-primary-600 hover:underline dark:text-primary-400">
              {{ $t('nav.tours') }}
            </NuxtLink>
            <NuxtLink :to="localePath(`/local-guides?destination=${destination}`)" class="text-primary-600 hover:underline dark:text-primary-400">
              {{ $t('nav.localGuides') }}
            </NuxtLink>
          </div>
        </div>
      </div>
    </section>

    <section v-if="topHotels.length > 0" class="mx-auto max-w-4xl px-4 pb-14">
      <div class="mb-6 flex items-center justify-between">
        <h2 class="text-xl font-bold text-text">{{ $t('home.topHotels') }}</h2>
        <NuxtLink :to="localePath('/hotels')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
          {{ $t('home.viewAllHotels') }}
        </NuxtLink>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <HotelCard v-for="hotel in topHotels" :key="hotel.id" :hotel="hotel" />
      </div>
    </section>

    <section class="mx-auto max-w-4xl px-4 pb-16">
      <div class="mb-6 flex items-center justify-between">
        <h2 class="text-xl font-bold text-text">{{ $t('home.latestNews') }}</h2>
        <NuxtLink
          :to="localePath('/blog')"
          class="text-sm text-primary-600 hover:underline dark:text-primary-400"
        >
          {{ $t('home.viewAllPosts') }}
        </NuxtLink>
      </div>

      <div v-if="latestPosts.length > 0" class="flex flex-col gap-4">
        <UCard v-for="post in latestPosts" :key="post.id">
          <NuxtLink :to="localePath(`/blog/${post.slug}`)" class="flex flex-col gap-4 sm:flex-row">
            <img
              v-if="post.coverImageUrl"
              :src="`${apiBaseUrl}${post.coverImageUrl}`"
              :alt="post.title"
              class="h-40 w-full rounded-lg object-cover sm:w-56"
            />
            <div class="flex flex-col justify-center">
              <h3 class="text-lg font-semibold text-text">{{ post.title }}</h3>
              <p v-if="post.excerpt" class="mt-2 text-text-muted">{{ post.excerpt }}</p>
            </div>
          </NuxtLink>
        </UCard>
      </div>
      <p v-else class="text-text-muted">{{ $t('blog.empty') }}</p>
    </section>
  </div>
</template>
