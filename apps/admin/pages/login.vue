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
  <div
    class="relative flex min-h-screen items-center justify-center bg-steel-50 text-steel-900 dark:bg-steel-950 dark:text-steel-100"
  >
    <div class="absolute right-4 top-4">
      <ThemeToggle />
    </div>

    <form
      class="w-full max-w-sm rounded-lg border border-steel-200 bg-white p-6 shadow-sm dark:border-steel-800 dark:bg-steel-900"
      @submit="onSubmit"
    >
      <h1 class="mb-1 flex items-center gap-2 text-lg font-semibold">
        <span class="inline-block h-5 w-1.5 rounded-full bg-brand-500" aria-hidden="true" />
        <span>As<span class="text-brand-600 dark:text-brand-400">Di</span> Cargo</span>
      </h1>
      <p class="mb-4 text-sm text-steel-500 dark:text-steel-400">Вход в админку</p>

      <UFormField label="Пароль" :error="errors.password">
        <UInput v-model="password" v-bind="passwordAttrs" type="password" placeholder="Пароль" class="w-full" />
      </UFormField>
      <p v-if="errorMessage" class="mt-2 text-sm text-error-500 dark:text-error-400">{{ errorMessage }}</p>
      <UButton class="mt-4 w-full justify-center" type="submit" :loading="loading">Войти</UButton>
    </form>
  </div>
</template>
