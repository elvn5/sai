<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { BlogPost, Hotel, ListingStats, LocalGuide, Tour, Trail } from '~/shared/api/types';
import { formatDateRange } from '~/shared/lib/format-date-range';
import { bishkekToday, usePartnerStats } from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.nav.listings'), robots: 'noindex' });

const toursQuery = useQuery({
  queryKey: ['tours', 'mine'],
  queryFn: () => api.get<Tour[]>('/tours/mine'),
  enabled: auth.isPartner,
});
const hotelsQuery = useQuery({
  queryKey: ['my-hotels'],
  queryFn: () => api.get<Hotel[]>('/hotels/mine'),
  enabled: auth.isPartner,
});
const guideQuery = useQuery({
  queryKey: ['my-local-guide'],
  queryFn: () => api.get<{ guide: LocalGuide | null }>('/local-guides/mine'),
  enabled: auth.isPartner,
});
const trailsQuery = useQuery({
  queryKey: ['trails', 'mine'],
  queryFn: () => api.get<Trail[]>('/trails/mine'),
  enabled: auth.isPartner,
});
const postsQuery = useQuery({
  queryKey: ['my-blog-posts'],
  queryFn: () => api.get<BlogPost[]>('/blog/mine'),
  enabled: auth.isPartner,
});
const statsQuery = usePartnerStats(ref(30));

const today = bishkekToday();
const tours = computed(() => toursQuery.data.value ?? []);
// Upcoming tours soonest first; past ones most recent first.
const upcomingTours = computed(() =>
  tours.value
    .filter((tour: Tour) => tour.endDate >= today)
    .sort((a: Tour, b: Tour) => a.startDate.localeCompare(b.startDate)),
);
const pastTours = computed(() => tours.value.filter((tour: Tour) => tour.endDate < today));
const hotels = computed(() => hotelsQuery.data.value ?? []);
const guide = computed(() => guideQuery.data.value?.guide ?? null);
const trails = computed(() => trailsQuery.data.value ?? []);
const posts = computed(() => postsQuery.data.value ?? []);

// Views over the last 30 days, by listing id.
const views = computed(() => {
  const map = new Map<string, number>();
  for (const item of statsQuery.data.value?.listings ?? []) {
    map.set((item as ListingStats).listing.id, (item as ListingStats).views);
  }
  return map;
});

function tourPath(tour: Tour): string {
  return `${tour.locale !== 'ru' ? `/${tour.locale}` : ''}/tours/${tour.slug}`;
}

const deleteTourMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/tours/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['tours'] });
    queryClient.invalidateQueries({ queryKey: ['partner', 'stats'] });
  },
});

function deleteTour(tour: Tour): void {
  if (window.confirm(t('cabinet.listings.deleteTourConfirm', { title: tour.title }))) {
    deleteTourMutation.mutate(tour.id);
  }
}

const deletePostMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/blog/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['my-blog-posts'] });
    queryClient.invalidateQueries({ queryKey: ['blog'] });
  },
});

function deletePost(post: BlogPost): void {
  if (window.confirm(t('profile.listings.deletePostConfirm', { title: post.title }))) {
    deletePostMutation.mutate(post.id);
  }
}
</script>

<template>
  <div class="flex flex-col gap-8">
    <div>
      <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.nav.listings') }}</h1>
      <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.listings.subtitle') }}</p>
    </div>

    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-2">
        <h2 class="font-semibold text-text">{{ $t('cabinet.listings.tours') }}</h2>
        <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/tours/new')">
          {{ $t('tours.addButton') }}
        </UButton>
      </div>
      <template v-for="group in [{ key: 'upcoming', items: upcomingTours }, { key: 'past', items: pastTours }]" :key="group.key">
        <p v-if="group.key === 'past' && group.items.length > 0" class="mt-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          {{ $t('cabinet.listings.pastTours') }}
        </p>
        <div
          v-for="tour in group.items"
          :key="tour.id"
          class="flex flex-col gap-3 rounded-card border border-border bg-surface px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
          :class="group.key === 'past' ? 'opacity-75' : ''"
        >
          <div class="min-w-0">
            <NuxtLink :to="tourPath(tour)" class="block truncate font-medium text-text hover:underline">{{ tour.title }}</NuxtLink>
            <p class="text-xs text-text-muted">
              {{ formatDateRange(locale, tour.startDate, tour.endDate) }} ·
              {{ $t('cabinet.listings.views30', { count: views.get(tour.id) ?? 0 }) }} ·
              {{ $t('reviews.count', { count: tour.reviewCount }) }}
            </p>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/partner/tours/${tour.id}`)">
              {{ $t('listing.edit') }}
            </UButton>
            <UButton
              size="sm"
              color="gray"
              icon="i-heroicons-document-duplicate"
              :to="localePath({ path: '/tours/new', query: { from: tour.id } })"
            >
              {{ $t('cabinet.listings.duplicate') }}
            </UButton>
            <UButton
              size="sm"
              color="gray"
              variant="ghost"
              icon="i-heroicons-trash"
              :aria-label="$t('tours.deleteButton')"
              :loading="deleteTourMutation.isPending.value && deleteTourMutation.variables.value === tour.id"
              @click="deleteTour(tour)"
            />
          </div>
        </div>
      </template>
      <p
        v-if="tours.length === 0 && toursQuery.isFetched.value"
        class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted"
      >
        {{ $t('cabinet.listings.noTours') }}
      </p>
    </section>

    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-2">
        <h2 class="font-semibold text-text">{{ $t('profile.listings.hotelsHeading') }}</h2>
        <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/hotels/new')">
          {{ $t('hotels.addButton') }}
        </UButton>
      </div>
      <div
        v-for="hotel in hotels"
        :key="hotel.id"
        class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3"
      >
        <NuxtLink :to="localePath(`/hotels/${hotel.slug}`)" class="min-w-0">
          <span class="block truncate font-medium text-text hover:underline">{{ hotel.name }}</span>
          <span class="block text-xs text-text-muted">
            {{ $t(`destinations.${hotel.destination}`) }} ·
            {{ $t('cabinet.listings.views30', { count: views.get(hotel.id) ?? 0 }) }} ·
            {{ $t('reviews.count', { count: hotel.reviewCount }) }}
          </span>
        </NuxtLink>
        <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/hotels/${hotel.slug}/edit`)">
          {{ $t('listing.edit') }}
        </UButton>
      </div>
      <p
        v-if="hotels.length === 0 && hotelsQuery.isFetched.value"
        class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted"
      >
        {{ $t('profile.listings.noHotels') }}
      </p>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('profile.listings.guideHeading') }}</h2>
      <div v-if="guide" class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3">
        <NuxtLink :to="localePath(`/local-guides/${guide.slug}`)" class="min-w-0">
          <span class="block truncate font-medium text-text hover:underline">{{ guide.name }}</span>
          <span class="block text-xs text-text-muted">
            {{ $t('cabinet.listings.views30', { count: views.get(guide.id) ?? 0 }) }} ·
            {{ $t('reviews.count', { count: guide.reviewCount }) }}
          </span>
        </NuxtLink>
        <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/local-guides/${guide.slug}/edit`)">
          {{ $t('listing.edit') }}
        </UButton>
      </div>
      <div
        v-else-if="guideQuery.isFetched.value"
        class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center"
      >
        <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.listings.guidePitch') }}</p>
        <UButton color="primary" :to="localePath('/local-guides/new')">{{ $t('localGuides.addButton') }}</UButton>
      </div>
    </section>

    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-2">
        <h2 class="font-semibold text-text">{{ $t('cabinet.listings.trails') }}</h2>
        <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/trails/new')">
          {{ $t('cabinet.listings.addTrail') }}
        </UButton>
      </div>
      <div
        v-for="trail in trails"
        :key="trail.id"
        class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3"
      >
        <NuxtLink :to="localePath(`/trails/${trail.slug}`)" class="min-w-0 truncate font-medium text-text hover:underline">
          {{ trail.name }}
        </NuxtLink>
        <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/trails/${trail.slug}/edit`)">
          {{ $t('listing.edit') }}
        </UButton>
      </div>
      <p
        v-if="trails.length === 0 && trailsQuery.isFetched.value"
        class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted"
      >
        {{ $t('cabinet.listings.noTrails') }}
      </p>
    </section>

    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-2">
        <h2 class="font-semibold text-text">{{ $t('profile.listings.postsHeading') }}</h2>
        <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/blog/new')">
          {{ $t('blog.form.newTitle') }}
        </UButton>
      </div>
      <div
        v-for="post in posts"
        :key="post.id"
        class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3"
      >
        <div class="min-w-0">
          <NuxtLink
            v-if="post.isPublished"
            :to="localePath(`/blog/${post.slug}`)"
            class="block truncate font-medium text-text hover:underline"
          >
            {{ post.title }}
          </NuxtLink>
          <span v-else class="block truncate font-medium text-text">{{ post.title }}</span>
          <span class="block text-xs text-text-muted">
            {{ post.isPublished ? $t('profile.listings.published') : $t('profile.listings.draft') }}
          </span>
        </div>
        <div class="flex shrink-0 gap-2">
          <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/blog/edit/${post.id}`)">
            {{ $t('listing.edit') }}
          </UButton>
          <UButton
            size="sm"
            color="gray"
            variant="ghost"
            icon="i-heroicons-trash"
            :aria-label="$t('profile.listings.deletePost')"
            @click="deletePost(post)"
          />
        </div>
      </div>
      <p
        v-if="posts.length === 0 && postsQuery.isFetched.value"
        class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted"
      >
        {{ $t('profile.listings.noPosts') }}
      </p>
    </section>
  </div>
</template>
