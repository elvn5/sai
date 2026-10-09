<script setup lang="ts">
import { useInfiniteQuery, useQuery } from '@tanstack/vue-query';
import LocalGuideCard from '~/components/entities/local-guide/LocalGuideCard.vue';
import { useApiClient } from '~/shared/api/client';
import type { LocalGuide, Paginated } from '~/shared/api/types';
import {
  DESTINATION_OPTIONS,
  GUIDE_LANGUAGE_OPTIONS,
  GUIDE_SPECIALIZATION_OPTIONS,
  MIN_RATING_OPTIONS,
} from '~/shared/lib/listing-options';
import { queryString, toQueryParams } from '~/shared/lib/listing-query';

const PAGE_SIZE = 12;
const SEARCH_DEBOUNCE_MS = 300;

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const api = useApiClient();
const auth = useAuth();

const filters = reactive({
  destination: queryString(route.query, 'destination') ?? '',
  language: queryString(route.query, 'language') ?? '',
  specialization: queryString(route.query, 'specialization') ?? '',
  maxPrice: queryString(route.query, 'maxPrice') ?? '',
  minRating: queryString(route.query, 'minRating') ?? '',
  hasVehicle: queryString(route.query, 'hasVehicle') === 'true',
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

const guidesQuery = useInfiniteQuery({
  queryKey: computed(() => ['local-guides', params.value] as const),
  queryFn: ({ pageParam }) => {
    const query = new URLSearchParams({
      ...params.value,
      sort: filters.sort,
      limit: String(PAGE_SIZE),
      offset: String(pageParam),
    });
    return api.get<Paginated<LocalGuide>>(`/local-guides?${query.toString()}`);
  },
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) => {
    const loaded = allPages.reduce((sum, page) => sum + page.items.length, 0);
    return loaded < lastPage.total ? loaded : undefined;
  },
});
await guidesQuery.suspense();

const guides = computed(() => guidesQuery.data.value?.pages.flatMap((page) => page.items) ?? []);
const total = computed(() => guidesQuery.data.value?.pages[0]?.total ?? 0);

// Only signed-in users without a profile yet get the "become a guide" CTA.
const myGuideQuery = useQuery({
  queryKey: ['my-local-guide'],
  queryFn: () => api.get<{ guide: LocalGuide | null }>('/local-guides/mine'),
  enabled: computed(() => Boolean(auth.user.value)),
});
const myGuide = computed(() => myGuideQuery.data.value?.guide ?? null);

function withAll(allKey: string, options: { value: string; labelKey: string }[]) {
  return [
    { value: '', label: t(allKey) },
    ...options.map((o) => ({ value: o.value, label: t(o.labelKey) })),
  ];
}
const destinationOptions = computed(() =>
  withAll('listing.filters.allDestinations', DESTINATION_OPTIONS),
);
const languageOptions = computed(() => withAll('localGuides.anyLanguage', GUIDE_LANGUAGE_OPTIONS));
const specializationOptions = computed(() =>
  withAll('localGuides.anySpecialization', GUIDE_SPECIALIZATION_OPTIONS),
);
const ratingOptions = computed(() => [
  { value: '', label: t('listing.filters.anyRating') },
  ...MIN_RATING_OPTIONS.map((value) => ({ value: String(value), label: `${value}+ ★` })),
]);
const sortOptions = computed(() =>
  ['rating', 'price_asc', 'price_desc', 'experience', 'newest'].map((value) => ({
    value,
    label: t(`listing.sort.${value}`),
  })),
);

const activeFilterCount = computed(
  () =>
    [
      filters.destination,
      filters.language,
      filters.specialization,
      filters.maxPrice,
      filters.minRating,
      filters.hasVehicle,
    ].filter(Boolean).length,
);

function resetFilters(): void {
  Object.assign(filters, {
    destination: '',
    language: '',
    specialization: '',
    maxPrice: '',
    minRating: '',
    hasVehicle: false,
  });
  searchInput.value = '';
}

useSeoMeta({
  title: () => t('localGuides.heading'),
  description: () => t('localGuides.metaDescription'),
  ogTitle: () => t('localGuides.heading'),
  ogDescription: () => t('localGuides.metaDescription'),
});
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl font-extrabold text-text">{{ $t('localGuides.heading') }}</h1>
        <p class="mt-1 text-text-muted">{{ $t('localGuides.subheading') }}</p>
      </div>
      <UButton
        v-if="auth.user.value && myGuide"
        color="gray"
        icon="i-heroicons-user-circle"
        :to="localePath(`/local-guides/${myGuide.slug}`)"
      >
        {{ $t('localGuides.myProfile') }}
      </UButton>
      <UButton
        v-else-if="auth.isPartner.value && myGuideQuery.isFetched.value"
        color="primary"
        icon="i-heroicons-plus"
        :to="localePath('/local-guides/new')"
      >
        {{ $t('localGuides.addButton') }}
      </UButton>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row">
      <UInput
        v-model="searchInput"
        icon="i-heroicons-magnifying-glass"
        :placeholder="$t('localGuides.searchPlaceholder')"
        class="flex-1"
      />
      <USelect v-model="filters.destination" :options="destinationOptions" option-attribute="label" class="sm:w-56" />
      <USelect v-model="filters.language" :options="languageOptions" option-attribute="label" class="sm:w-44" />
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
          <UFormGroup :label="$t('localGuides.specializationsLabel')">
            <USelect v-model="filters.specialization" :options="specializationOptions" option-attribute="label" />
          </UFormGroup>
          <UFormGroup :label="$t('listing.filters.maxPricePerDay')" :hint="$t('listing.currency')">
            <UInput v-model="filters.maxPrice" type="number" min="0" step="1000" />
          </UFormGroup>
          <UFormGroup :label="$t('listing.filters.minRating')">
            <USelect v-model="filters.minRating" :options="ratingOptions" option-attribute="label" />
          </UFormGroup>
          <UCheckbox v-model="filters.hasVehicle" :label="$t('localGuides.vehicleFilter')" />
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

        <div v-if="guides.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <LocalGuideCard v-for="guide in guides" :key="guide.id" :guide="guide" />
        </div>
        <div
          v-else
          class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center"
        >
          <UIcon name="i-heroicons-user-group" class="h-10 w-10 text-text-muted" />
          <p class="text-text-muted">{{ $t('localGuides.empty') }}</p>
          <UButton v-if="activeFilterCount > 0" color="gray" @click="resetFilters">
            {{ $t('listing.filters.reset') }}
          </UButton>
        </div>

        <div v-if="guidesQuery.hasNextPage.value" class="mt-6 text-center">
          <UButton color="gray" :loading="guidesQuery.isFetchingNextPage.value" @click="guidesQuery.fetchNextPage()">
            {{ $t('listing.loadMore') }}
          </UButton>
        </div>
      </div>
    </div>
  </section>
</template>
