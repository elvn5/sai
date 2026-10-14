<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import type { TourFormPayload } from '~/components/features/tour-form/form-payload';
import { CancellationPolicy, Destination, type Tour } from '~/shared/api/types';
import { contactSchemas, emptyToNull, optionalNumber } from '~/shared/lib/form-schemas';
import { CANCELLATION_OPTIONS, DESTINATION_OPTIONS } from '~/shared/lib/listing-options';

// Creating, editing and duplicating a tour. `initial` prefills the fields
// (a duplicate passes the original without its dates).
const props = defineProps<{
  initial?: Partial<Tour> | null;
  submitting: boolean;
  submitError: string | null;
  submitLabel: string;
}>();

const emit = defineEmits<{ submit: [payload: TourFormPayload] }>();

const { t } = useI18n();

const cover = ref<string[]>(props.initial?.coverImageUrl ? [props.initial.coverImageUrl] : []);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: toTypedSchema(
    z
      .object({
        title: z.string().trim().min(1, t('login.required')).max(255),
        excerpt: z.string().max(500).optional(),
        content: z.string().min(1, t('login.required')).max(20000),
        price: optionalNumber(z.number().int().nonnegative()),
        durationDays: optionalNumber(z.number().int().positive()),
        location: z.string().max(255).optional(),
        destination: z.nativeEnum(Destination).optional(),
        cancellationPolicy: z.nativeEnum(CancellationPolicy),
        ...contactSchemas({
          phone: t('listing.form.invalidPhone'),
          telegram: t('listing.form.invalidTelegram'),
        }),
        startDate: z.string().min(1, t('login.required')),
        endDate: z.string().min(1, t('login.required')),
      })
      .refine((data) => data.endDate >= data.startDate, {
        message: t('tours.dateRangeInvalid'),
        path: ['endDate'],
      }),
  ),
  initialValues: {
    title: props.initial?.title ?? '',
    excerpt: props.initial?.excerpt ?? '',
    content: props.initial?.content ?? '',
    price: props.initial?.price ?? undefined,
    durationDays: props.initial?.durationDays ?? undefined,
    location: props.initial?.location ?? '',
    destination: props.initial?.destination ?? undefined,
    cancellationPolicy: props.initial?.cancellationPolicy ?? CancellationPolicy.FLEXIBLE,
    startDate: props.initial?.startDate ?? '',
    endDate: props.initial?.endDate ?? '',
    phone: props.initial?.phone ?? '',
    whatsapp: props.initial?.whatsapp ?? '',
    telegram: props.initial?.telegram ?? '',
  },
});

const [title, titleAttrs] = defineField('title');
const [excerpt, excerptAttrs] = defineField('excerpt');
const [content] = defineField('content');
const [price, priceAttrs] = defineField('price');
const [durationDays, durationDaysAttrs] = defineField('durationDays');
const [location, locationAttrs] = defineField('location');
const [startDate, startDateAttrs] = defineField('startDate');
const [endDate, endDateAttrs] = defineField('endDate');
const [destination] = defineField('destination');
const [cancellationPolicy] = defineField('cancellationPolicy');
const [phone, phoneAttrs] = defineField('phone');
const [whatsapp, whatsappAttrs] = defineField('whatsapp');
const [telegram, telegramAttrs] = defineField('telegram');

const destinationOptions = computed(() =>
  DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
);
const cancellationOptions = computed(() =>
  CANCELLATION_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) })),
);

const onSubmit = handleSubmit((values) => {
  emit('submit', {
    title: values.title,
    excerpt: emptyToNull(values.excerpt),
    content: values.content,
    price: values.price ?? null,
    durationDays: values.durationDays ?? null,
    location: emptyToNull(values.location),
    startDate: values.startDate,
    endDate: values.endDate,
    destination: values.destination ?? null,
    cancellationPolicy: values.cancellationPolicy,
    coverImageUrl: cover.value[0] ?? null,
    phone: emptyToNull(values.phone),
    whatsapp: emptyToNull(values.whatsapp),
    telegram: emptyToNull(values.telegram)?.replace(/^@/, '') ?? null,
  });
});
</script>

<template>
  <form class="flex flex-col gap-6" @submit="onSubmit">
    <UFormGroup :label="$t('tours.titleLabel')" :error="errors.title">
      <UInput v-model="title" v-bind="titleAttrs" />
    </UFormGroup>

    <UFormGroup :label="$t('tours.excerptLabel')" :error="errors.excerpt">
      <UInput v-model="excerpt" v-bind="excerptAttrs" />
    </UFormGroup>

    <div class="grid grid-cols-1 gap-6 sm:grid-cols-3">
      <UFormGroup :label="$t('tours.priceLabel')" :error="errors.price">
        <UInput v-model.number="price" v-bind="priceAttrs" type="number" min="0" />
      </UFormGroup>

      <UFormGroup :label="$t('tours.durationLabel')" :error="errors.durationDays">
        <UInput v-model.number="durationDays" v-bind="durationDaysAttrs" type="number" min="1" />
      </UFormGroup>

      <UFormGroup :label="$t('tours.locationLabel')" :error="errors.location">
        <UInput v-model="location" v-bind="locationAttrs" />
      </UFormGroup>
    </div>

    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
      <UFormGroup :label="$t('tours.startDateLabel')" :error="errors.startDate">
        <UInput v-model="startDate" v-bind="startDateAttrs" type="date" />
      </UFormGroup>

      <UFormGroup :label="$t('tours.endDateLabel')" :error="errors.endDate">
        <UInput v-model="endDate" v-bind="endDateAttrs" type="date" />
      </UFormGroup>
    </div>

    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
      <UFormGroup :label="$t('listing.form.destinationLabel')" :error="errors.destination">
        <USelect
          v-model="destination"
          :options="destinationOptions"
          option-attribute="label"
          :placeholder="$t('listing.form.choose')"
        />
      </UFormGroup>

      <UFormGroup :label="$t('listing.form.cancellationLabel')" :error="errors.cancellationPolicy">
        <USelect v-model="cancellationPolicy" :options="cancellationOptions" option-attribute="label" />
      </UFormGroup>
    </div>

    <div class="grid grid-cols-1 gap-6 sm:grid-cols-3">
      <UFormGroup :label="$t('listing.form.whatsappLabel')" :error="errors.whatsapp">
        <UInput v-model="whatsapp" v-bind="whatsappAttrs" placeholder="+996 555 123 456" />
      </UFormGroup>
      <UFormGroup :label="$t('listing.form.telegramLabel')" :error="errors.telegram">
        <UInput v-model="telegram" v-bind="telegramAttrs" placeholder="@username" />
      </UFormGroup>
      <UFormGroup :label="$t('listing.form.phoneLabel')" :error="errors.phone">
        <UInput v-model="phone" v-bind="phoneAttrs" placeholder="+996 555 123 456" />
      </UFormGroup>
    </div>

    <UFormGroup :label="$t('tours.coverLabel')">
      <PhotoUploader v-model="cover" :max="1" folder="tours" class="[&>div]:!grid-cols-1" />
    </UFormGroup>

    <UFormGroup :label="$t('tours.contentLabel')" :error="errors.content">
      <TiptapEditor v-model="content" :placeholder="$t('tours.contentPlaceholder')" />
    </UFormGroup>

    <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

    <UButton type="submit" color="primary" block :loading="submitting">
      {{ submitLabel }}
    </UButton>
  </form>
</template>
