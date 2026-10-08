<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import type { LocalGuideFormPayload } from '~/components/features/local-guide-form/form-payload';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import ChipToggleGroup from '~/components/shared/ui/ChipToggleGroup.vue';
import type { LocalGuide } from '~/shared/api/types';
import { contactSchemas, emptyToNull, optionalNumber } from '~/shared/lib/form-schemas';
import {
  DESTINATION_OPTIONS,
  GUIDE_LANGUAGE_OPTIONS,
  GUIDE_SPECIALIZATION_OPTIONS,
} from '~/shared/lib/listing-options';

const MAX_PHOTOS = 20;

const props = defineProps<{
  initial?: LocalGuide | null;
  defaultName?: string;
  submitting: boolean;
  submitError: string | null;
  submitLabel: string;
}>();

const emit = defineEmits<{ submit: [payload: LocalGuideFormPayload] }>();

const { t } = useI18n();

const avatar = ref<string[]>(props.initial?.avatarUrl ? [props.initial.avatarUrl] : []);
const photos = ref<string[]>(props.initial?.photos ?? []);
const languages = ref<string[]>(props.initial?.languages ?? []);
const specializations = ref<string[]>(props.initial?.specializations ?? []);
const destinations = ref<string[]>(props.initial?.destinations ?? []);
const listErrors = ref<{ languages?: string; specializations?: string; destinations?: string }>({});

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z.string().trim().min(1, t('login.required')).max(255),
      headline: z.string().trim().max(160).optional(),
      bio: z.string().min(1, t('login.required')).max(20000),
      pricePerDay: optionalNumber(z.number().int().nonnegative()),
      experienceYears: optionalNumber(z.number().int().min(0).max(80)),
      hasVehicle: z.boolean(),
      ...contactSchemas({
        phone: t('listing.form.invalidPhone'),
        telegram: t('listing.form.invalidTelegram'),
      }),
    }),
  ),
  initialValues: {
    name: props.initial?.name ?? props.defaultName ?? '',
    headline: props.initial?.headline ?? '',
    bio: props.initial?.bio ?? '',
    pricePerDay: props.initial?.pricePerDay ?? undefined,
    experienceYears: props.initial?.experienceYears ?? undefined,
    hasVehicle: props.initial?.hasVehicle ?? false,
    phone: props.initial?.phone ?? '',
    whatsapp: props.initial?.whatsapp ?? '',
    telegram: props.initial?.telegram ?? '',
  },
});

const [name, nameAttrs] = defineField('name');
const [headline, headlineAttrs] = defineField('headline');
const [bio] = defineField('bio');
const [pricePerDay, priceAttrs] = defineField('pricePerDay');
const [experienceYears, experienceAttrs] = defineField('experienceYears');
const [hasVehicle] = defineField('hasVehicle');
const [phone, phoneAttrs] = defineField('phone');
const [whatsapp, whatsappAttrs] = defineField('whatsapp');
const [telegram, telegramAttrs] = defineField('telegram');

// The chip groups live outside vee-validate (they're plain refs), so their
// "at least one" rule is checked here alongside the schema.
function validateLists(): boolean {
  const required = t('listing.form.pickAtLeastOne');
  listErrors.value = {
    languages: languages.value.length === 0 ? required : undefined,
    specializations: specializations.value.length === 0 ? required : undefined,
    destinations: destinations.value.length === 0 ? required : undefined,
  };
  return Object.values(listErrors.value).every((error) => !error);
}

const onSubmit = handleSubmit(
  (values) => {
    if (!validateLists()) return;
    emit('submit', {
      name: values.name,
      headline: emptyToNull(values.headline),
      bio: values.bio,
      avatarUrl: avatar.value[0] ?? null,
      coverImageUrl: photos.value[0] ?? null,
      photos: photos.value,
      languages: languages.value,
      specializations: specializations.value,
      destinations: destinations.value,
      pricePerDay: values.pricePerDay ?? null,
      experienceYears: values.experienceYears ?? null,
      hasVehicle: values.hasVehicle,
      phone: emptyToNull(values.phone),
      whatsapp: emptyToNull(values.whatsapp),
      telegram: emptyToNull(values.telegram)?.replace(/^@/, '') ?? null,
    });
  },
  () => {
    validateLists();
  },
);
</script>

<template>
  <form class="flex flex-col gap-8" @submit="onSubmit">
    <fieldset class="flex flex-col gap-5">
      <legend class="mb-3 text-lg font-semibold text-text">{{ $t('listing.form.basics') }}</legend>
      <div class="flex flex-col gap-5 sm:flex-row">
        <UFormGroup :label="$t('localGuides.avatarLabel')" class="sm:w-32">
          <PhotoUploader v-model="avatar" :max="1" folder="guides" class="[&>div]:!grid-cols-1" />
        </UFormGroup>
        <div class="flex flex-1 flex-col gap-5">
          <UFormGroup :label="$t('localGuides.nameLabel')" :error="errors.name" required>
            <UInput v-model="name" v-bind="nameAttrs" :maxlength="255" />
          </UFormGroup>
          <UFormGroup :label="$t('localGuides.headlineLabel')" :error="errors.headline">
            <UInput
              v-model="headline"
              v-bind="headlineAttrs"
              :maxlength="160"
              :placeholder="$t('localGuides.headlinePlaceholder')"
            />
          </UFormGroup>
        </div>
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <UFormGroup :label="$t('localGuides.priceLabel')" :error="errors.pricePerDay" :hint="$t('listing.currency')">
          <UInput v-model.number="pricePerDay" v-bind="priceAttrs" type="number" min="0" />
        </UFormGroup>
        <UFormGroup :label="$t('localGuides.experienceLabel')" :error="errors.experienceYears">
          <UInput v-model.number="experienceYears" v-bind="experienceAttrs" type="number" min="0" max="80" />
        </UFormGroup>
      </div>
      <UCheckbox v-model="hasVehicle" :label="$t('localGuides.hasVehicleLabel')" />
    </fieldset>

    <fieldset class="flex flex-col gap-5">
      <legend class="mb-3 text-lg font-semibold text-text">{{ $t('localGuides.expertise') }}</legend>
      <UFormGroup :label="$t('localGuides.languagesLabel')" :error="listErrors.languages" required>
        <ChipToggleGroup v-model="languages" :options="GUIDE_LANGUAGE_OPTIONS" />
      </UFormGroup>
      <UFormGroup :label="$t('localGuides.specializationsLabel')" :error="listErrors.specializations" required>
        <ChipToggleGroup v-model="specializations" :options="GUIDE_SPECIALIZATION_OPTIONS" />
      </UFormGroup>
      <UFormGroup :label="$t('localGuides.destinationsLabel')" :error="listErrors.destinations" required>
        <ChipToggleGroup v-model="destinations" :options="DESTINATION_OPTIONS" />
      </UFormGroup>
    </fieldset>

    <fieldset class="flex flex-col gap-3">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('localGuides.bioLabel') }}</legend>
      <TiptapEditor v-model="bio" :placeholder="$t('localGuides.bioPlaceholder')" />
      <p v-if="errors.bio" class="text-sm text-red-600">{{ errors.bio }}</p>
    </fieldset>

    <fieldset class="flex flex-col gap-3">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('listing.photos.heading') }}</legend>
      <p class="text-sm text-text-muted">{{ $t('localGuides.photosHint') }}</p>
      <PhotoUploader v-model="photos" :max="MAX_PHOTOS" folder="guides" show-cover-badge />
    </fieldset>

    <fieldset class="flex flex-col gap-5">
      <legend class="mb-1 text-lg font-semibold text-text">{{ $t('listing.form.contacts') }}</legend>
      <p class="-mt-2 text-sm text-text-muted">{{ $t('listing.form.contactsHint') }}</p>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
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
    </fieldset>

    <p v-if="submitError" class="text-sm text-red-600">{{ submitError }}</p>
    <UButton type="submit" color="primary" size="lg" block :loading="submitting">{{ submitLabel }}</UButton>
  </form>
</template>
