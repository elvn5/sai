<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import { useApiClient } from '~/shared/api/client';
import { CancellationPolicy, Destination } from '~/shared/api/types';
import type { Tour } from '~/shared/api/types';
import { contactSchemas, emptyToNull } from '~/shared/lib/form-schemas';
import { CANCELLATION_OPTIONS, DESTINATION_OPTIONS } from '~/shared/lib/listing-options';
import { slugify } from '~/shared/lib/slugify';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const queryClient = useQueryClient();

const coverFile = ref<File | null>(null);
const submitError = ref<string | null>(null);

const formSchema = toTypedSchema(
  z
    .object({
      title: z.string().min(1, t('login.required')).max(255),
      excerpt: z.string().max(500).optional(),
      content: z.string().min(1, t('login.required')).max(20000),
      price: z.number().int().nonnegative().optional(),
      durationDays: z.number().int().positive().optional(),
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
);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: formSchema,
  initialValues: { content: '', cancellationPolicy: CancellationPolicy.FLEXIBLE },
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

useSeoMeta({
  title: () => t('tours.newTitle'),
  robots: 'noindex',
});

useRequirePartner();

function onCoverChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  coverFile.value = target.files?.[0] ?? null;
}

const createMutation = useMutation({
  mutationFn: async (values: {
    title: string;
    excerpt?: string;
    content: string;
    price?: number;
    durationDays?: number;
    location?: string;
    startDate: string;
    endDate: string;
    destination?: Destination;
    cancellationPolicy: CancellationPolicy;
    phone?: string;
    whatsapp?: string;
    telegram?: string;
  }) => {
    let coverImageUrl: string | null = null;
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('file', coverFile.value);
      const upload = await api.upload<{ url: string }>('/uploads/image', formData);
      coverImageUrl = upload.url;
    }

    return api.post<Tour>('/tours', {
      locale: locale.value,
      slug: slugify(values.title),
      title: values.title,
      excerpt: values.excerpt || null,
      content: values.content,
      price: values.price ?? null,
      durationDays: values.durationDays ?? null,
      location: values.location || null,
      startDate: values.startDate,
      endDate: values.endDate,
      destination: values.destination ?? null,
      cancellationPolicy: values.cancellationPolicy,
      phone: emptyToNull(values.phone),
      whatsapp: emptyToNull(values.whatsapp),
      telegram: emptyToNull(values.telegram)?.replace(/^@/, '') ?? null,
      coverImageUrl,
    });
  },
  onSuccess: (tour) => {
    queryClient.invalidateQueries({ queryKey: ['tours'] });
    navigateTo(localePath(`/tours/${tour.slug}`));
  },
  onError: () => {
    submitError.value = t('tours.createError');
  },
});

const onSubmit = handleSubmit((values) => {
  submitError.value = null;
  createMutation.mutate(values);
});
</script>

<template>
  <section class="mx-auto max-w-2xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('tours.newTitle') }}</h1>

    <form class="mt-8 flex flex-col gap-6" @submit="onSubmit">
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
        <input
          type="file"
          accept="image/*"
          class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
          @change="onCoverChange"
        />
      </UFormGroup>

      <UFormGroup :label="$t('tours.contentLabel')" :error="errors.content">
        <TiptapEditor v-model="content" :placeholder="$t('tours.contentPlaceholder')" />
      </UFormGroup>

      <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

      <UButton type="submit" color="primary" block :loading="createMutation.isPending.value">
        {{ $t('tours.submit') }}
      </UButton>
    </form>
  </section>
</template>
