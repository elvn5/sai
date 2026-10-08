<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import type { HotelFormPayload } from '~/components/features/hotel-form/form-payload';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import ChipToggleGroup from '~/components/shared/ui/ChipToggleGroup.vue';
import { CancellationPolicy, Destination, type Hotel, HotelType } from '~/shared/api/types';
import { contactSchemas, emptyToNull, optionalNumber } from '~/shared/lib/form-schemas';
import {
  CANCELLATION_OPTIONS,
  DESTINATION_OPTIONS,
  HOTEL_AMENITY_OPTIONS,
  HOTEL_TYPE_OPTIONS,
} from '~/shared/lib/listing-options';

const MAX_PHOTOS = 20;

const props = defineProps<{
  initial?: Hotel | null;
  submitting: boolean;
  submitError: string | null;
  submitLabel: string;
}>();

const emit = defineEmits<{ submit: [payload: HotelFormPayload] }>();

const { t } = useI18n();

const photos = ref<string[]>(props.initial?.photos ?? []);
const amenities = ref<string[]>(props.initial?.amenities ?? []);

const typeOptions = computed(() =>
  HOTEL_TYPE_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
);
const destinationOptions = computed(() =>
  DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
);
const starOptions = computed(() => [
  { value: '', label: t('hotels.noStars') },
  ...[1, 2, 3, 4, 5].map((n) => ({ value: n, label: '★'.repeat(n) })),
]);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z.string().trim().min(1, t('login.required')).max(255),
      type: z.nativeEnum(HotelType),
      destination: z.nativeEnum(Destination, { required_error: t('login.required') }),
      description: z.string().min(1, t('login.required')).max(20000),
      address: z.string().max(255).optional(),
      latitude: optionalNumber(z.number().min(-90).max(90)),
      longitude: optionalNumber(z.number().min(-180).max(180)),
      priceFrom: optionalNumber(z.number().int().nonnegative()),
      stars: optionalNumber(z.number().int().min(1).max(5)),
      cancellationPolicy: z.nativeEnum(CancellationPolicy),
      checkInTime: z.string().optional(),
      checkOutTime: z.string().optional(),
      website: z.string().url(t('listing.form.invalidUrl')).optional().or(z.literal('')),
      ...contactSchemas({
        phone: t('listing.form.invalidPhone'),
        telegram: t('listing.form.invalidTelegram'),
      }),
    }),
  ),
  initialValues: {
    name: props.initial?.name ?? '',
    type: props.initial?.type ?? HotelType.GUESTHOUSE,
    destination: props.initial?.destination,
    description: props.initial?.description ?? '',
    address: props.initial?.address ?? '',
    latitude: props.initial?.latitude ?? undefined,
    longitude: props.initial?.longitude ?? undefined,
    priceFrom: props.initial?.priceFrom ?? undefined,
    stars: props.initial?.stars ?? undefined,
    cancellationPolicy: props.initial?.cancellationPolicy ?? CancellationPolicy.FLEXIBLE,
    checkInTime: props.initial?.checkInTime ?? '14:00',
    checkOutTime: props.initial?.checkOutTime ?? '12:00',
    website: props.initial?.website ?? '',
    phone: props.initial?.phone ?? '',
    whatsapp: props.initial?.whatsapp ?? '',
    telegram: props.initial?.telegram ?? '',
  },
});

const [name, nameAttrs] = defineField('name');
const [type] = defineField('type');
const [destination] = defineField('destination');
const [description] = defineField('description');
const [address, addressAttrs] = defineField('address');
const [latitude, latitudeAttrs] = defineField('latitude');
const [longitude, longitudeAttrs] = defineField('longitude');
const [priceFrom, priceFromAttrs] = defineField('priceFrom');
const [stars] = defineField('stars');
const [cancellationPolicy] = defineField('cancellationPolicy');
const [checkInTime, checkInAttrs] = defineField('checkInTime');
const [checkOutTime, checkOutAttrs] = defineField('checkOutTime');
const [website, websiteAttrs] = defineField('website');
const [phone, phoneAttrs] = defineField('phone');
const [whatsapp, whatsappAttrs] = defineField('whatsapp');
const [telegram, telegramAttrs] = defineField('telegram');

const onSubmit = handleSubmit((values) => {
  emit('submit', {
    name: values.name,
    type: values.type,
    destination: values.destination,
    description: values.description,
    address: emptyToNull(values.address),
    latitude: values.latitude ?? null,
    longitude: values.longitude ?? null,
    priceFrom: values.priceFrom ?? null,
    stars: values.stars ?? null,
    amenities: amenities.value,
    cancellationPolicy: values.cancellationPolicy,
    checkInTime: emptyToNull(values.checkInTime),
    checkOutTime: emptyToNull(values.checkOutTime),
    coverImageUrl: photos.value[0] ?? null,
    photos: photos.value,
    phone: emptyToNull(values.phone),
    whatsapp: emptyToNull(values.whatsapp),
    telegram: emptyToNull(values.telegram)?.replace(/^@/, '') ?? null,
    website: emptyToNull(values.website),
  });
});
</script>

<template>
  <form class="flex flex-col gap-8" @submit="onSubmit">
    <fieldset class="flex flex-col gap-5">
      <legend class="mb-3 text-lg font-semibold text-text">{{ $t('listing.form.basics') }}</legend>
      <UFormGroup :label="$t('hotels.nameLabel')" :error="errors.name" required>
        <UInput v-model="name" v-bind="nameAttrs" :maxlength="255" />
      </UFormGroup>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
        <UFormGroup :label="$t('hotels.typeLabel')" :error="errors.type" required>
          <USelect v-model="type" :options="typeOptions" option-attribute="label" />
        </UFormGroup>
        <UFormGroup :label="$t('hotels.starsLabel')" :error="errors.stars">
          <USelect v-model="stars" :options="starOptions" option-attribute="label" />
        </UFormGroup>
        <UFormGroup :label="$t('hotels.priceLabel')" :error="errors.priceFrom" :hint="$t('listing.currency')">
          <UInput v-model.number="priceFrom" v-bind="priceFromAttrs" type="number" min="0" />
        </UFormGroup>
      </div>
    </fieldset>

    <fieldset class="flex flex-col gap-3">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('listing.photos.heading') }}</legend>
      <p class="text-sm text-text-muted">{{ $t('listing.photos.hint') }}</p>
      <PhotoUploader v-model="photos" :max="MAX_PHOTOS" folder="hotels" show-cover-badge />
    </fieldset>

    <fieldset class="flex flex-col gap-5">
      <legend class="mb-3 text-lg font-semibold text-text">{{ $t('listing.form.location') }}</legend>
      <UFormGroup :label="$t('listing.form.destinationLabel')" :error="errors.destination" required>
        <USelect
          v-model="destination"
          :options="destinationOptions"
          option-attribute="label"
          :placeholder="$t('listing.form.choose')"
        />
      </UFormGroup>
      <UFormGroup :label="$t('listing.form.addressLabel')" :error="errors.address">
        <UInput v-model="address" v-bind="addressAttrs" :maxlength="255" />
      </UFormGroup>
      <div class="grid grid-cols-2 gap-5">
        <UFormGroup :label="$t('listing.form.latitudeLabel')" :error="errors.latitude">
          <UInput v-model.number="latitude" v-bind="latitudeAttrs" type="number" step="any" placeholder="42.4907" />
        </UFormGroup>
        <UFormGroup :label="$t('listing.form.longitudeLabel')" :error="errors.longitude">
          <UInput v-model.number="longitude" v-bind="longitudeAttrs" type="number" step="any" placeholder="78.3936" />
        </UFormGroup>
      </div>
      <p class="-mt-2 text-xs text-text-muted">{{ $t('listing.form.coordsHint') }}</p>
    </fieldset>

    <fieldset class="flex flex-col gap-5">
      <legend class="mb-3 text-lg font-semibold text-text">{{ $t('hotels.conditions') }}</legend>
      <UFormGroup :label="$t('hotels.amenitiesLabel')">
        <ChipToggleGroup v-model="amenities" :options="HOTEL_AMENITY_OPTIONS" />
      </UFormGroup>
      <UFormGroup :label="$t('listing.form.cancellationLabel')" :error="errors.cancellationPolicy" required>
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <label
            v-for="option in CANCELLATION_OPTIONS"
            :key="option.value"
            class="flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors"
            :class="cancellationPolicy === option.value ? 'border-brand-500 bg-brand-500/5' : 'border-border'"
          >
            <input v-model="cancellationPolicy" type="radio" :value="option.value" class="mt-1" />
            <span>
              <span class="block text-sm font-medium text-text">{{ $t(option.labelKey) }}</span>
              <span class="block text-xs text-text-muted">{{ $t(`listing.cancellationHint.${option.value}`) }}</span>
            </span>
          </label>
        </div>
      </UFormGroup>
      <div class="grid grid-cols-2 gap-5">
        <UFormGroup :label="$t('hotels.checkInLabel')">
          <UInput v-model="checkInTime" v-bind="checkInAttrs" type="time" />
        </UFormGroup>
        <UFormGroup :label="$t('hotels.checkOutLabel')">
          <UInput v-model="checkOutTime" v-bind="checkOutAttrs" type="time" />
        </UFormGroup>
      </div>
    </fieldset>

    <fieldset class="flex flex-col gap-3">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('hotels.descriptionLabel') }}</legend>
      <TiptapEditor v-model="description" :placeholder="$t('hotels.descriptionPlaceholder')" />
      <p v-if="errors.description" class="text-sm text-red-600">{{ errors.description }}</p>
    </fieldset>

    <fieldset class="flex flex-col gap-5">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('listing.form.contacts') }}</legend>
      <p class="-mt-2 text-sm text-text-muted">{{ $t('listing.form.contactsHint') }}</p>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <UFormGroup :label="$t('listing.form.whatsappLabel')" :error="errors.whatsapp">
          <UInput v-model="whatsapp" v-bind="whatsappAttrs" placeholder="+996 555 123 456" icon="i-heroicons-chat-bubble-left-right" />
        </UFormGroup>
        <UFormGroup :label="$t('listing.form.telegramLabel')" :error="errors.telegram">
          <UInput v-model="telegram" v-bind="telegramAttrs" placeholder="@username" icon="i-heroicons-paper-airplane" />
        </UFormGroup>
        <UFormGroup :label="$t('listing.form.phoneLabel')" :error="errors.phone">
          <UInput v-model="phone" v-bind="phoneAttrs" placeholder="+996 312 000 000" icon="i-heroicons-phone" />
        </UFormGroup>
        <UFormGroup :label="$t('listing.form.websiteLabel')" :error="errors.website">
          <UInput v-model="website" v-bind="websiteAttrs" placeholder="https://" icon="i-heroicons-globe-alt" />
        </UFormGroup>
      </div>
    </fieldset>

    <p v-if="submitError" class="text-sm text-red-600">{{ submitError }}</p>
    <UButton type="submit" color="primary" size="lg" block :loading="submitting">{{ submitLabel }}</UButton>
  </form>
</template>
