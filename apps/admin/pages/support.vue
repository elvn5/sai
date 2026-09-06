<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { SupportInfo } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const form = reactive({ message: '', phone: '', telegramContact: '', email: '', workingHours: '' });
const loading = ref(false);
const saving = ref(false);
const saved = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  try {
    const data = await api.get<SupportInfo>('/support');
    form.message = data.message;
    form.phone = data.phone ?? '';
    form.telegramContact = data.telegramContact ?? '';
    form.email = data.email ?? '';
    form.workingHours = data.workingHours ?? '';
  } finally {
    loading.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  saved.value = false;
  try {
    await api.put('/support', {
      message: form.message,
      phone: form.phone || null,
      telegramContact: form.telegramContact || null,
      email: form.email || null,
      workingHours: form.workingHours || null,
    });
    saved.value = true;
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="max-w-xl">
    <h1 class="mb-4 text-xl font-semibold">Поддержка</h1>

    <div class="flex flex-col gap-3 rounded-lg border border-steel-200 bg-white p-4 dark:border-steel-800 dark:bg-steel-950">
      <UFormField label="Сообщение (показывается в боте)">
        <UTextarea v-model="form.message" :rows="4" class="w-full" />
      </UFormField>
      <UFormField label="Телефон">
        <UInput v-model="form.phone" class="w-full" />
      </UFormField>
      <UFormField label="Telegram-контакт">
        <UInput v-model="form.telegramContact" placeholder="@username" class="w-full" />
      </UFormField>
      <UFormField label="Email">
        <UInput v-model="form.email" class="w-full" />
      </UFormField>
      <UFormField label="Часы работы">
        <UInput v-model="form.workingHours" class="w-full" />
      </UFormField>

      <div class="mt-2 flex items-center gap-2">
        <UButton :loading="saving" @click="save">Сохранить</UButton>
        <span v-if="saved" class="text-sm text-primary-600 dark:text-primary-400">Сохранено</span>
      </div>
    </div>
  </div>
</template>
