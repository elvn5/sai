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

    <div class="flex flex-col gap-3 rounded-lg border bg-white p-4">
      <UFormGroup label="Сообщение (показывается в боте)">
        <UTextarea v-model="form.message" :rows="4" />
      </UFormGroup>
      <UFormGroup label="Телефон">
        <UInput v-model="form.phone" />
      </UFormGroup>
      <UFormGroup label="Telegram-контакт">
        <UInput v-model="form.telegramContact" placeholder="@username" />
      </UFormGroup>
      <UFormGroup label="Email">
        <UInput v-model="form.email" />
      </UFormGroup>
      <UFormGroup label="Часы работы">
        <UInput v-model="form.workingHours" />
      </UFormGroup>

      <div class="mt-2 flex items-center gap-2">
        <UButton :loading="saving" @click="save">Сохранить</UButton>
        <span v-if="saved" class="text-sm text-green-600">Сохранено</span>
      </div>
    </div>
  </div>
</template>
