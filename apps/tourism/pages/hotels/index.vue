<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query';
import HotelCard from '~/components/entities/hotel/HotelCard.vue';
import ChipToggleGroup from '~/components/shared/ui/ChipToggleGroup.vue';
import { useApiClient } from '~/shared/api/client';
import type { Hotel, Paginated } from '~/shared/api/types';
import {
  DESTINATION_OPTIONS,
  HOTEL_AMENITY_OPTIONS,
  HOTEL_TYPE_OPTIONS,
  MIN_RATING_OPTIONS,
} from '~/shared/lib/listing-options';
import { queryList, queryString, toQueryParams } from '~/shared/lib/listing-query';

const PAGE_SIZE = 12;
const SEARCH_DEBOUNCE_MS = 300;

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const api = useApiClient();
const auth = useAuth();

// Filters are initialized from — and written back to — the URL, so
// /hotels?destination=karakol links from the home page work and filtered
// views can be shared.
const filters = reactive({
  destination: queryString(route.query, 'destination') ?? '',
  type: queryString(route.query, 'type') ?? '',
  maxPrice: queryString(route.query, 'maxPrice') ?? '',
  minRating: queryString(route.query, 'minRating') ?? '',
  amenities: queryList(route.query, 'amenities'),
  freeCancellation: queryString(route.query, 'freeCancellation') === 'true',
  sort: queryString(route.query, 'sort') ?? 'rating',
});
const searchInput = ref(queryString(route.query, 'search') ?? '');
const search = ref(searchInput.value);
const filtersOpen = ref(false);

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(searchInput, (value: string) => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    search.value = value.trim();
  }, SEARCH_DEBOUNCE_MS);
});

const params = computed(() =>
  toQueryParams({
    ...filters,
    search: search.value,
    sort: filters.sort === 'rating' ? '' : filters.sort,
  }),
);

watch(params, (next: Record<string, string>) => {
  void router.replace({ query: next });
});

const hotelsQuery = useInfiniteQuery({
  queryKey: computed(() => ['hotels', params.value] as const),
  queryFn: ({ pageParam }) => {
    const query = new URLSearchParams({
      ...params.value,
      sort: filters.sort,
      limit: String(PAGE_SIZE),
      offset: String(pageParam),
    });
    return api.get<Paginated<Hotel>>(`/hotels?${query.toString()}`);
  },
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) => {
    const loaded = allPages.reduce((sum, page) => sum + page.items.length, 0);
    return loaded < lastPage.total ? loaded : undefined;
  },
});
await hotelsQuery.suspense();

const hotels = computed(() => hotelsQuery.data.value?.pages.flatMap((page) => page.items) ?? []);
const total = computed(() => hotelsQuery.data.value?.pages[0]?.total ?? 0);

const destinationOptions = computed(() => [
  { value: '', label: t('listing.filters.allDestinations') },
  ...DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
]);
const typeOptions = computed(() => [
  { value: '', label: t('listing.filters.allTypes') },
  ...HOTEL_TYPE_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
]);
const ratingOptions = computed(() => [
  { value: '', label: t('listing.filters.anyRating') },
  ...MIN_RATING_OPTIONS.map((value) => ({ value: String(value), label: `${value}+ ★` })),
]);
const sortOptions = computed(() =>
  ['rating', 'price_asc', 'price_desc', 'newest'].map((value) => ({
    value,
    label: t(`listing.sort.${value}`),
  })),
);

const activeFilterCount = computed(
  () =>
    [
      filters.destination,
      filters.type,
      filters.maxPrice,
      filters.minRating,
      filters.freeCancellation,
    ].filter(Boolean).length + filters.amenities.length,
);

function resetFilters(): void {
  Object.assign(filters, {
    destination: '',
    type: '',
    maxPrice: '',
    minRating: '',
    amenities: [],
    freeCancellation: false,
  });
  searchInput.value = '';
}

const destinationName = computed(() =>
  filters.destination ? t(`destinations.${filters.destination}`) : null,
);

useSeoMeta({
  title: () =>
    destinationName.value
      ? t('hotels.headingIn', { place: destinationName.value })
      : t('hotels.heading'),
  description: () => t('hotels.metaDescription'),
  ogTitle: () => t('hotels.heading'),
  ogDescription: () => t('hotels.metaDescription'),
});
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl font-extrabold text-text">
          {{ destinationName ? $t('hotels.headingIn', { place: destinationName }) : $t('hotels.heading') }}
        </h1>
        <p class="mt-1 text-text-muted">{{ $t('hotels.subheading') }}</p>
      </div>
      <UButton
        v-if="auth.user.value"
        color="primary"
        icon="i-heroicons-plus"
        :to="localePath('/hotels/new')"
      >
        {{ $t('hotels.addButton') }}
      </UButton>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row">
      <UInput
        v-model="searchInput"
        icon="i-heroicons-magnifying-glass"
        :placeholder="$t('hotels.searchPlaceholder')"
        class="flex-1"
      />
      <USelect v-model="filters.destination" :options="destinationOptions" option-attribute="label" class="sm:w-56" />
      <UButton
        color="gray"
        icon="i-heroicons-adjustments-horizontal"
        class="lg:hidden"
        @click="filtersOpen = !filtersOpen"
      >
        {{ $t('listing.filters.title') }}<template v-if="activeFilterCount">&nbsp;({{ activeFilterCount }})</template>
      </UButton>
    </div>

    <div class="mt-6 flex flex-col gap-8 lg:flex-row">
      <aside class="lg:block lg:w-64 lg:shrink-0" :class="filtersOpen ? 'block' : 'hidden'">
        <div class="flex flex-col gap-5 rounded-card border border-border bg-surface p-4 lg:sticky lg:top-20">
          <UFormGroup :label="$t('hotels.typeLabel')">
            <USelect v-model="filters.type" :options="typeOptions" option-attribute="label" />
          </UFormGroup>
          <UFormGroup :label="$t('listing.filters.maxPrice')" :hint="$t('listing.currency')">
            <UInput v-model="filters.maxPrice" type="number" min="0" step="500" />
          </UFormGroup>
          <UFormGroup :label="$t('listing.filters.minRating')">
            <USelect v-model="filters.minRating" :options="ratingOptions" option-attribute="label" />
          </UFormGroup>
          <UCheckbox v-model="filters.freeCancellation" :label="$t('listing.filters.freeCancellation')" />
          <UFormGroup :label="$t('hotels.amenitiesLabel')">
            <ChipToggleGroup v-model="filters.amenities" :options="HOTEL_AMENITY_OPTIONS.slice(0, 8)" />
          </UFormGroup>
          <UButton v-if="activeFilterCount > 0" color="gray" variant="ghost" @click="resetFilters">
            {{ $t('listing.filters.reset') }}
          </UButton>
        </div>
      </aside>

      <div class="flex-1">
        <div class="mb-4 flex items-center justify-between gap-4">
          <p class="text-sm text-text-muted">{{ $t('listing.found', { count: total }) }}</p>
          <USelect v-model="filters.sort" :options="sortOptions" option-attribute="label" size="sm" class="w-48" />
        </div>

        <div v-if="hotels.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <HotelCard v-for="hotel in hotels" :key="hotel.id" :hotel="hotel" />
        </div>
        <div
          v-else
          class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center"
        >
          <UIcon name="i-heroicons-building-office-2" class="h-10 w-10 text-text-muted" />
          <p class="text-text-muted">{{ $t('hotels.empty') }}</p>
          <UButton v-if="activeFilterCount > 0" color="gray" @click="resetFilters">
            {{ $t('listing.filters.reset') }}
          </UButton>
        </div>

        <div v-if="hotelsQuery.hasNextPage.value" class="mt-6 text-center">
          <UButton color="gray" :loading="hotelsQuery.isFetchingNextPage.value" @click="hotelsQuery.fetchNextPage()">
            {{ $t('listing.loadMore') }}
          </UButton>
        </div>
      </div>
    </div>
  </section>
</template>
