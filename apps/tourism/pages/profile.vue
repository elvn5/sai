<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import OrderCard from '~/components/entities/order/OrderCard.vue';
import OutingCard from '~/components/entities/outing/OutingCard.vue';
import AwardCodeCard from '~/components/features/award-code/AwardCodeCard.vue';
import TelegramLink from '~/components/features/telegram-link/TelegramLink.vue';
import { useApiClient } from '~/shared/api/client';
import { PartnerStatus } from '~/shared/api/types';
import type {
  AppUser,
  BlogPost,
  Hotel,
  LocalGuide,
  MyAchievements,
  Order,
  Outing,
} from '~/shared/api/types';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuth();
const api = useApiClient();
const queryClient = useQueryClient();
const { isTma, getInitData } = useTelegramWebApp();

const route = useRoute();

// Listings are for partners only (they alone publish).
const tabs = computed(() => [
  { key: 'info', label: t('profile.tabs.info'), icon: 'i-heroicons-user-circle' },
  { key: 'achievements', label: t('profile.tabs.achievements'), icon: 'i-mdi-trophy-outline' },
  { key: 'bookings', label: t('profile.tabs.bookings'), icon: 'i-mdi-ticket-confirmation-outline' },
  ...(auth.isPartner.value
    ? [
        {
          key: 'listings',
          label: t('profile.tabs.listings'),
          icon: 'i-heroicons-building-storefront',
        },
      ]
    : []),
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
const myHotelsQuery = useQuery({
  queryKey: ['my-hotels'],
  queryFn: () => api.get<Hotel[]>('/hotels/mine'),
  enabled: auth.isPartner,
});
const myGuideQuery = useQuery({
  queryKey: ['my-local-guide'],
  queryFn: () => api.get<{ guide: LocalGuide | null }>('/local-guides/mine'),
  enabled: auth.isPartner,
});
const myPostsQuery = useQuery({
  queryKey: ['my-blog-posts'],
  queryFn: () => api.get<BlogPost[]>('/blog/mine'),
  enabled: auth.isPartner,
});
const myOrdersQuery = useQuery({
  queryKey: ['orders', 'mine'],
  queryFn: () => api.get<Order[]>('/orders/mine'),
  enabled: isSignedIn,
});
const receivedOrdersQuery = useQuery({
  queryKey: ['orders', 'received'],
  queryFn: () => api.get<Order[]>('/orders/received'),
  enabled: auth.isPartner,
});
const myOrders = computed(() => myOrdersQuery.data.value ?? []);
const receivedOrders = computed(() => receivedOrdersQuery.data.value ?? []);

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
const myHotels = computed(() => myHotelsQuery.data.value ?? []);
const myGuide = computed(() => myGuideQuery.data.value?.guide ?? null);
const myPosts = computed(() => myPostsQuery.data.value ?? []);
const myAchievements = computed(() => myAchievementsQuery.data.value ?? null);

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

        <div v-else-if="item.key === 'listings'" class="mt-6 flex flex-col gap-8">
          <div>
            <h2 class="mb-3 font-semibold text-text">{{ $t('booking.received') }}</h2>
            <div v-if="receivedOrders.length > 0" class="flex flex-col gap-2">
              <OrderCard v-for="order in receivedOrders" :key="order.id" :order="order" for-partner />
            </div>
            <p v-else class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted">
              {{ $t('booking.noReceived') }}
            </p>
          </div>

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

          <div>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="font-semibold text-text">{{ $t('profile.listings.postsHeading') }}</h2>
              <UButton size="sm" color="primary" icon="i-heroicons-plus" :to="localePath('/blog/new')">
                {{ $t('blog.form.newTitle') }}
              </UButton>
            </div>
            <div v-if="myPosts.length > 0" class="flex flex-col gap-2">
              <div
                v-for="post in myPosts"
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
            </div>
            <p v-else class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted">
              {{ $t('profile.listings.noPosts') }}
            </p>
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
              <UButton color="primary" icon="i-mdi-medal" :to="localePath('/achievements/award')">
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
