<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import { PartnerStatus } from '~/shared/api/types';
import type { AppUser, Hotel, LocalGuide } from '~/shared/api/types';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuth();
const api = useApiClient();
const queryClient = useQueryClient();
const { isTma, getInitData } = useTelegramWebApp();

const tabs = [
  { key: 'info', label: t('profile.tabs.info'), icon: 'i-heroicons-user-circle' },
  { key: 'listings', label: t('profile.tabs.listings'), icon: 'i-heroicons-building-storefront' },
  { key: 'history', label: t('profile.tabs.history'), icon: 'i-heroicons-map' },
  { key: 'partner', label: t('profile.tabs.partner'), icon: 'i-heroicons-briefcase' },
];

const isSignedIn = computed(() => Boolean(auth.user.value));
const myHotelsQuery = useQuery({
  queryKey: ['my-hotels'],
  queryFn: () => api.get<Hotel[]>('/hotels/mine'),
  enabled: isSignedIn,
});
const myGuideQuery = useQuery({
  queryKey: ['my-local-guide'],
  queryFn: () => api.get<{ guide: LocalGuide | null }>('/local-guides/mine'),
  enabled: isSignedIn,
});
const myHotels = computed(() => myHotelsQuery.data.value ?? []);
const myGuide = computed(() => myGuideQuery.data.value?.guide ?? null);

const applyMutation = useMutation({
  mutationFn: () => api.post<{ user: AppUser }>('/partners/apply'),
  onSuccess: (res) => {
    queryClient.setQueryData(['auth', 'me'], res.user);
  },
});

useSeoMeta({
  title: () => t('profile.title'),
  robots: 'noindex',
});

const displayName = computed(
  () => auth.user.value?.firstName ?? auth.user.value?.username ?? auth.user.value?.email ?? '',
);

async function handleLogout(): Promise<void> {
  await auth.logout();
  await navigateTo(localePath('/'));
}

onMounted(async () => {
  if (!auth.checked.value) {
    await auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
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

        <div v-else-if="item.key === 'listings'" class="mt-6 flex flex-col gap-8">
          <div>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="font-semibold text-text">{{ $t('profile.listings.guideHeading') }}</h2>
            </div>
            <div
              v-if="myGuide"
              class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3"
            >
              <NuxtLink :to="localePath(`/local-guides/${myGuide.slug}`)" class="font-medium text-text hover:underline">
                {{ myGuide.name }}
              </NuxtLink>
              <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/local-guides/${myGuide.slug}/edit`)">
                {{ $t('listing.edit') }}
              </UButton>
            </div>
            <div
              v-else
              class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center"
            >
              <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.listings.guidePitch') }}</p>
              <UButton color="primary" :to="localePath('/local-guides/new')">{{ $t('localGuides.addButton') }}</UButton>
            </div>
          </div>

          <div>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="font-semibold text-text">{{ $t('profile.listings.hotelsHeading') }}</h2>
              <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/hotels/new')">
                {{ $t('hotels.addButton') }}
              </UButton>
            </div>
            <div v-if="myHotels.length > 0" class="flex flex-col gap-2">
              <div
                v-for="hotel in myHotels"
                :key="hotel.id"
                class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3"
              >
                <NuxtLink :to="localePath(`/hotels/${hotel.slug}`)" class="min-w-0">
                  <span class="block truncate font-medium text-text hover:underline">{{ hotel.name }}</span>
                  <span class="block text-xs text-text-muted">
                    {{ $t(`destinations.${hotel.destination}`) }} · {{ $t('reviews.count', { count: hotel.reviewCount }) }}
                  </span>
                </NuxtLink>
                <UButton size="sm" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/hotels/${hotel.slug}/edit`)">
                  {{ $t('listing.edit') }}
                </UButton>
              </div>
            </div>
            <p v-else class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted">
              {{ $t('profile.listings.noHotels') }}
            </p>
          </div>
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

        <div v-else-if="item.key === 'partner'" class="mt-6">
          <div
            v-if="auth.user.value.partnerStatus === PartnerStatus.APPROVED"
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-heroicons-check-badge" class="h-10 w-10 text-brand-600 dark:text-brand-400" />
            <p class="font-medium text-text">{{ $t('profile.partner.approved') }}</p>
            <UButton color="primary" class="mt-2" @click="navigateTo(localePath('/tours/new'))">
              {{ $t('profile.partner.addTourCta') }}
            </UButton>
          </div>

          <div
            v-else-if="auth.user.value.partnerStatus === PartnerStatus.PENDING"
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-heroicons-clock" class="h-10 w-10 text-text-muted" />
            <p class="font-medium text-text">{{ $t('profile.partner.pending') }}</p>
          </div>

          <div
            v-else
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-heroicons-briefcase" class="h-10 w-10 text-text-muted" />
            <p v-if="auth.user.value.partnerStatus === PartnerStatus.REJECTED" class="font-medium text-text">
              {{ $t('profile.partner.rejected') }}
            </p>
            <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.partner.pitch') }}</p>
            <UButton
              color="primary"
              class="mt-2"
              :loading="applyMutation.isPending.value"
              @click="applyMutation.mutate()"
            >
              {{ $t('profile.partner.applyCta') }}
            </UButton>
          </div>
        </div>
      </template>
    </UTabs>
  </section>
</template>
