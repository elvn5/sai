<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import OrderCard from '~/components/entities/order/OrderCard.vue';
import OutingCard from '~/components/entities/outing/OutingCard.vue';
import AwardCodeCard from '~/components/features/award-code/AwardCodeCard.vue';
import TelegramLink from '~/components/features/telegram-link/TelegramLink.vue';
import { useApiClient } from '~/shared/api/client';
import { PartnerStatus } from '~/shared/api/types';
import type { AppUser, MyAchievements, Order, Outing } from '~/shared/api/types';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuth();
const api = useApiClient();
const queryClient = useQueryClient();
const { isTma, getInitData } = useTelegramWebApp();

const route = useRoute();

// Partners manage their listings and bookings in the cabinet (/partner).
const tabs = computed(() => [
  { key: 'info', label: t('profile.tabs.info'), icon: 'i-heroicons-user-circle' },
  { key: 'achievements', label: t('profile.tabs.achievements'), icon: 'i-mdi-trophy-outline' },
  { key: 'bookings', label: t('profile.tabs.bookings'), icon: 'i-mdi-ticket-confirmation-outline' },
  { key: 'history', label: t('profile.tabs.history'), icon: 'i-heroicons-map' },
  { key: 'partner', label: t('profile.tabs.partner'), icon: 'i-heroicons-briefcase' },
]);
// ?tab=partner (etc.) opens that tab.
const selectedTab = computed({
  get: () =>
    Math.max(
      0,
      tabs.value.findIndex((tab: { key: string }) => tab.key === route.query.tab),
    ),
  set: (index: number) => {
    void navigateTo({ query: { ...route.query, tab: tabs.value[index]?.key } }, { replace: true });
  },
});

const isSignedIn = computed(() => Boolean(auth.user.value));
const myOrdersQuery = useQuery({
  queryKey: ['orders', 'mine'],
  queryFn: () => api.get<Order[]>('/orders/mine'),
  enabled: isSignedIn,
});
const myOrders = computed(() => myOrdersQuery.data.value ?? []);

const myOutingsQuery = useQuery({
  queryKey: ['outings', 'mine'],
  queryFn: () => api.get<Outing[]>('/outings/mine'),
  enabled: isSignedIn,
});
const myOutings = computed(() => myOutingsQuery.data.value ?? []);
const upcomingOutings = computed(() =>
  myOutings.value.filter((o: Outing) => new Date(o.startsAt) > new Date()).reverse(),
);
const pastOutings = computed(() =>
  myOutings.value.filter((o: Outing) => new Date(o.startsAt) <= new Date()),
);

const myAchievementsQuery = useQuery({
  queryKey: ['my-achievements'],
  queryFn: () => api.get<MyAchievements>('/achievements/me'),
  enabled: isSignedIn,
});
const myAchievements = computed(() => myAchievementsQuery.data.value ?? null);

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

    <UTabs v-model="selectedTab" :items="tabs" class="mt-8">
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
          <div class="flex flex-col gap-2 rounded-card border border-border bg-surface px-4 py-3">
            <span class="text-sm text-text-muted">{{ $t('profile.fields.telegram') }}</span>
            <TelegramLink />
          </div>

          <UButton color="gray" variant="soft" class="mt-2 self-start" @click="handleLogout">
            {{ $t('profile.logout') }}
          </UButton>
        </div>

        <div v-else-if="item.key === 'achievements'" class="mt-6 flex flex-col gap-4">
          <template v-if="myAchievements">
            <div class="flex items-center justify-between gap-4 rounded-card border border-border bg-surface px-4 py-3">
              <span class="text-sm text-text-muted">{{ $t('profile.achievements.summary', { count: myAchievements.awards.length }) }}</span>
              <span class="text-lg font-black text-amber-500">{{ $t('hallOfFame.points', { count: myAchievements.points }) }}</span>
            </div>
            <AwardCodeCard :code="myAchievements.code" />
            <p class="text-sm text-text-muted">{{ $t('hallOfFame.mine.howTo') }}</p>
          </template>
          <UButton color="primary" icon="i-mdi-trophy" class="self-start" :to="localePath('/hall-of-fame')">
            {{ $t('profile.achievements.open') }}
          </UButton>
        </div>

        <div v-else-if="item.key === 'bookings'" class="mt-6 flex flex-col gap-2">
          <OrderCard v-for="order in myOrders" :key="order.id" :order="order" />
          <div
            v-if="myOrders.length === 0"
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-mdi-ticket-confirmation-outline" class="h-10 w-10 text-text-muted" />
            <p class="font-medium text-text">{{ $t('booking.noBookings') }}</p>
            <UButton color="primary" class="mt-2" :to="localePath('/tours')">{{ $t('booking.browseTours') }}</UButton>
          </div>
        </div>

        <div v-else-if="item.key === 'history'" class="mt-6 flex flex-col gap-8">
          <div v-if="upcomingOutings.length > 0">
            <h2 class="mb-3 font-semibold text-text">{{ $t('profile.history.upcoming') }}</h2>
            <div class="flex flex-col gap-2">
              <OutingCard v-for="outing in upcomingOutings" :key="outing.id" :outing="outing" />
            </div>
          </div>
          <div v-if="pastOutings.length > 0">
            <h2 class="mb-3 font-semibold text-text">{{ $t('profile.history.past') }}</h2>
            <div class="flex flex-col gap-2">
              <OutingCard v-for="outing in pastOutings" :key="outing.id" :outing="outing" />
            </div>
          </div>
          <div
            v-if="myOutings.length === 0"
            class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center"
          >
            <UIcon name="i-heroicons-map" class="h-10 w-10 text-text-muted" />
            <p class="font-medium text-text">{{ $t('profile.history.empty') }}</p>
            <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.history.emptyHint') }}</p>
            <UButton color="primary" class="mt-2" @click="navigateTo(localePath('/outings'))">
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
            <div class="mt-2 flex flex-wrap justify-center gap-2">
              <UButton color="primary" icon="i-heroicons-squares-2x2" :to="localePath('/partner')">
                {{ $t('cabinet.open') }}
              </UButton>
              <UButton color="gray" icon="i-mdi-medal" :to="localePath('/achievements/award')">
                {{ $t('profile.partner.awardCta') }}
              </UButton>
              <UButton color="gray" icon="i-heroicons-map" :to="localePath('/tours/new')">
                {{ $t('profile.partner.addTourCta') }}
              </UButton>
              <UButton color="gray" icon="i-heroicons-pencil-square" :to="localePath('/blog/new')">
                {{ $t('blog.form.newTitle') }}
              </UButton>
            </div>
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
            <p class="max-w-sm text-sm text-text-muted">{{ $t('profile.partner.publishOnly') }}</p>
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
