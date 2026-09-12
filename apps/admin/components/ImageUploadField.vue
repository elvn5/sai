<template>
  <div class="flex items-center gap-3">
    <img
      v-if="modelValue"
      :src="fullUrl"
      class="h-20 w-20 rounded border border-gray-200 object-cover"
      alt=""
    />
    <div
      v-else
      class="flex h-20 w-20 items-center justify-center rounded border border-dashed border-gray-300 text-gray-400"
    >
      <UIcon name="i-heroicons-photo" class="h-6 w-6" />
    </div>
    <div class="flex flex-col gap-1">
      <UButton size="xs" color="gray" variant="soft" :loading="uploading" @click="fileInput?.click()">
        {{ modelValue ? 'Заменить' : 'Загрузить' }}
      </UButton>
      <UButton
        v-if="modelValue"
        size="xs"
        color="red"
        variant="ghost"
        @click="emit('update:modelValue', null)"
      >
        Удалить
      </UButton>
    </div>
    <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onSelect" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useApiClient } from '~/shared/api/client';

const props = defineProps<{ modelValue: string | null }>();
const emit = defineEmits<{ 'update:modelValue': [value: string | null] }>();

const api = useApiClient();
const config = useRuntimeConfig();
const fileInput = ref<HTMLInputElement | null>(null);
const uploading = ref(false);

const fullUrl = computed(() =>
  props.modelValue ? `${config.public.apiBaseUrl}${props.modelValue}` : '',
);

async function onSelect(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  uploading.value = true;
  try {
    const { url } = await api.upload<{ url: string }>('/uploads/image', file);
    emit('update:modelValue', url);
  } finally {
    uploading.value = false;
    target.value = '';
  }
}
</script>
