<script setup lang="ts">
import { useMutation } from '@tanstack/vue-query';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { AppUser, Order, Tour } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';

// Book places on a tour and pay online: the order is priced on the
// server; the customer is sent to the payment page.
const props = defineProps<{ tour: Tour }>();

const MAX_PLACES = 20;

const { t, locale } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const api = useApiClient();
const auth = useAuth();

const quantity = ref(1);
const contactName = ref(auth.user.value?.firstName ?? '');
const contactPhone = ref(auth.user.value?.phone ?? '');
const comment = ref('');
const error = ref<string | null>(null);

watch(
  () => auth.user.value,
  (user: AppUser | null) => {
    if (user && !contactName.value) contactName.value = user.firstName ?? '';
    if (user && !contactPhone.value) contactPhone.value = user.phone ?? '';
  },
);

const today = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Bishkek' });
const bookable = computed(
  () =>
    Boolean(props.tour.price && props.tour.price > 0 && props.tour.userId) &&
    props.tour.startDate >= today &&
    props.tour.userId !== auth.user.value?.id,
);
const total = computed(() => (props.tour.price ?? 0) * quantity.value);

const book = useMutation({
  mutationFn: () =>
    api.post<Order>('/orders', {
      kind: 'tour',
      itemId: props.tour.id,
      quantity: quantity.value,
      contactName: contactName.value.trim(),
      contactPhone: contactPhone.value.trim(),
      comment: comment.value.trim() || null,
    }),
  onSuccess: (order) => {
    if (order.paymentUrl) window.location.href = order.paymentUrl;
  },
  onError: (e) => {
    error.value =
      e instanceof ApiError && e.status === 400
        ? t('booking.checkFields')
        : e instanceof ApiError && e.status === 502
          ? t('booking.gatewayDown')
          : e instanceof ApiError && e.status === 409
            ? t('booking.unavailable')
            : t('booking.error');
  },
});

function submit(): void {
  error.value = null;
  if (!contactName.value.trim() || !contactPhone.value.trim()) {
    error.value = t('booking.checkFields');
    return;
  }
  book.mutate();
}
</script>

<template>
  <div v-if="bookable" class="rounded-card border-2 border-brand-500/40 bg-surface p-4">
    <h3 class="flex items-center gap-2 font-bold text-text">
      <UIcon name="i-mdi-ticket-confirmation-outline" class="h-5 w-5 text-brand-600 dark:text-brand-400" />
      {{ $t('booking.title') }}
    </h3>

    <template v-if="auth.user.value">
      <form class="mt-3 flex flex-col gap-3" @submit.prevent="submit">
        <div class="flex items-center justify-between gap-3">
          <span class="text-sm text-text">{{ $t('booking.places') }}</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" color="gray" icon="i-heroicons-minus" :disabled="quantity <= 1" :aria-label="$t('booking.less')" @click="quantity--" />
            <span class="w-6 text-center font-bold text-text">{{ quantity }}</span>
            <UButton size="xs" color="gray" icon="i-heroicons-plus" :disabled="quantity >= MAX_PLACES" :aria-label="$t('booking.more')" @click="quantity++" />
          </div>
        </div>
        <UInput v-model="contactName" :placeholder="$t('booking.name')" :maxlength="120" icon="i-heroicons-user" />
        <UInput v-model="contactPhone" type="tel" :placeholder="$t('booking.phone')" :maxlength="32" icon="i-heroicons-phone" />
        <UTextarea v-model="comment" :placeholder="$t('booking.comment')" :rows="2" :maxlength="1000" />
        <div class="flex items-baseline justify-between border-t border-border pt-3">
          <span class="text-sm text-text-muted">{{ $t('booking.total') }}</span>
          <span class="text-xl font-black text-text">{{ formatAmount(locale, total) }} {{ $t('listing.currency') }}</span>
        </div>
        <p v-if="error" class="text-sm text-brand-600">{{ error }}</p>
        <UButton type="submit" color="primary" size="lg" block icon="i-mdi-credit-card-outline" :loading="book.isPending.value">
          {{ $t('booking.pay') }}
        </UButton>
        <p class="flex items-center gap-1 text-xs text-text-muted">
          <UIcon name="i-mdi-shield-check-outline" class="h-4 w-4" />{{ $t('booking.secure') }}
        </p>
      </form>
    </template>
    <div v-else class="mt-3 flex flex-col gap-2">
      <p class="text-sm text-text-muted">{{ $t('booking.signIn') }}</p>
      <UButton color="primary" block :to="localePath({ path: '/login', query: { redirect: route.fullPath } })">{{ $t('booking.signInCta') }}</UButton>
    </div>
  </div>
</template>
