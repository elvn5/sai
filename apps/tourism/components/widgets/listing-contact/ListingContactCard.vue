<script setup lang="ts">
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import type { CancellationPolicy } from '~/shared/api/types';
import { phoneLink, telegramLink, whatsappLink } from '~/shared/lib/contact-links';

// The "booking" box on a listing page. Locally most bookings happen over
// WhatsApp/Telegram rather than a checkout, so contact is the primary CTA.
const props = defineProps<{
  listingName: string;
  priceLabel: string | null;
  ratingAverage: number;
  reviewCount: number;
  cancellationPolicy: CancellationPolicy | null;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
  website?: string | null;
}>();

const { t } = useI18n();

const whatsappHref = computed(() =>
  props.whatsapp
    ? whatsappLink(
        props.whatsapp,
        t('listing.contact.messageTemplate', { name: props.listingName }),
      )
    : null,
);
const hasContacts = computed(() =>
  Boolean(props.phone || props.whatsapp || props.telegram || props.website),
);
</script>

<template>
  <UCard>
    <div class="flex flex-col gap-4">
      <p v-if="priceLabel" class="text-2xl font-bold text-brand-600 dark:text-brand-400">{{ priceLabel }}</p>

      <a href="#reviews" class="flex items-center gap-2 text-sm">
        <RatingStars :value="ratingAverage" size="md" />
        <span v-if="reviewCount > 0" class="font-semibold text-text">{{ ratingAverage.toFixed(1) }}</span>
        <span class="text-text-muted">{{ $t('reviews.count', { count: reviewCount }) }}</span>
      </a>

      <div
        v-if="cancellationPolicy"
        class="flex items-start gap-2 rounded-lg px-3 py-2 text-sm"
        :class="
          cancellationPolicy === 'free'
            ? 'bg-green-500/10 text-green-700 dark:text-green-400'
            : 'bg-gray-500/10 text-text-muted'
        "
      >
        <UIcon
          :name="cancellationPolicy === 'free' ? 'i-heroicons-check-circle' : 'i-heroicons-information-circle'"
          class="mt-0.5 h-4 w-4 shrink-0"
        />
        <div>
          <p class="font-medium">{{ $t(`listing.cancellation.${cancellationPolicy}`) }}</p>
          <p class="text-xs opacity-80">{{ $t(`listing.cancellationHint.${cancellationPolicy}`) }}</p>
        </div>
      </div>

      <div v-if="hasContacts" class="flex flex-col gap-2">
        <UButton
          v-if="whatsappHref"
          :to="whatsappHref"
          target="_blank"
          block
          size="lg"
          icon="i-heroicons-chat-bubble-left-right"
          class="!bg-[#25D366] !text-white hover:!bg-[#1ebe5b]"
        >
          {{ $t('listing.contact.whatsapp') }}
        </UButton>
        <UButton
          v-if="telegram"
          :to="telegramLink(telegram)"
          target="_blank"
          block
          size="lg"
          icon="i-heroicons-paper-airplane"
          class="!bg-[#229ED9] !text-white hover:!bg-[#1c8cc2]"
        >
          {{ $t('listing.contact.telegram') }}
        </UButton>
        <UButton v-if="phone" :to="phoneLink(phone)" block color="gray" icon="i-heroicons-phone">
          {{ phone }}
        </UButton>
        <UButton
          v-if="website"
          :to="website"
          target="_blank"
          block
          color="gray"
          variant="ghost"
          icon="i-heroicons-globe-alt"
        >
          {{ $t('listing.contact.website') }}
        </UButton>
      </div>
      <p v-else class="text-sm text-text-muted">{{ $t('listing.contact.none') }}</p>
    </div>
  </UCard>
</template>
