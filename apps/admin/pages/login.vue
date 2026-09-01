<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import { ApiError } from '~/shared/api/client';
import { useAuthStore } from '~/stores/auth';

definePageMeta({ layout: 'default' });

const auth = useAuthStore();
const errorMessage = ref<string | null>(null);
const loading = ref(false);

const schema = toTypedSchema(
  z.object({
    password: z.string().min(1, 'Введите пароль'),
  }),
);

const { handleSubmit, defineField, errors } = useForm({ validationSchema: schema });
const [password, passwordAttrs] = defineField('password');

const onSubmit = handleSubmit(async (values) => {
  errorMessage.value = null;
  loading.value = true;
  try {
    await auth.login(values.password);
    await navigateTo('/');
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? 'Неверный пароль' : 'Ошибка сети';
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50">
    <form class="w-full max-w-sm rounded-lg border bg-white p-6 shadow-sm" @submit="onSubmit">
      <h1 class="mb-4 text-lg font-semibold">Вход в админку</h1>
      <UFormGroup label="Пароль" :error="errors.password">
        <UInput v-model="password" v-bind="passwordAttrs" type="password" placeholder="Пароль" />
      </UFormGroup>
      <p v-if="errorMessage" class="mt-2 text-sm text-red-600">{{ errorMessage }}</p>
      <UButton class="mt-4 w-full justify-center" type="submit" :loading="loading">Войти</UButton>
    </form>
  </div>
</template>
