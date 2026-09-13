<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import AppButton from '~/components/shared/ui/AppButton.vue';
import AppInput from '~/components/shared/ui/AppInput.vue';
import { ApiError } from '~/shared/api/client';
import type { TelegramWidgetPayload } from '~/shared/api/types';

const { t } = useI18n();
const localePath = useLocalePath();
const { isTma, getInitData } = useTelegramWebApp();
const auth = useAuth();
const config = useRuntimeConfig();

const mode = ref<'login' | 'register'>('login');
const identifierType = ref<'email' | 'phone'>('email');
const errorMessage = ref<string | null>(null);
const submitting = ref(false);
const tmaLoading = ref(false);
const widgetContainer = ref<HTMLDivElement | null>(null);

const loginSchema = toTypedSchema(
  z.object({
    identifier: z.string().min(1, t('login.required')),
    password: z.string().min(1, t('login.required')),
  }),
);

const registerSchema = toTypedSchema(
  z.object({
    identifier: z.string().min(1, t('login.required')),
    password: z.string().min(8, t('login.passwordTooShort')),
    firstName: z.string().optional(),
  }),
);

const {
  handleSubmit: handleLoginSubmit,
  defineField: defineLoginField,
  errors: loginErrors,
} = useForm({ validationSchema: loginSchema });
const [loginIdentifier, loginIdentifierAttrs] = defineLoginField('identifier');
const [loginPassword, loginPasswordAttrs] = defineLoginField('password');

const {
  handleSubmit: handleRegisterSubmit,
  defineField: defineRegisterField,
  errors: registerErrors,
} = useForm({ validationSchema: registerSchema });
const [registerIdentifier, registerIdentifierAttrs] = defineRegisterField('identifier');
const [registerPassword, registerPasswordAttrs] = defineRegisterField('password');
const [registerFirstName, registerFirstNameAttrs] = defineRegisterField('firstName');

const onLogin = handleLoginSubmit(async (values) => {
  errorMessage.value = null;
  submitting.value = true;
  try {
    await auth.login({ identifier: values.identifier, password: values.password });
    await navigateTo(localePath('/'));
  } catch (error) {
    errorMessage.value =
      error instanceof ApiError ? t('login.invalidCredentials') : t('login.networkError');
  } finally {
    submitting.value = false;
  }
});

const onRegister = handleRegisterSubmit(async (values) => {
  errorMessage.value = null;
  submitting.value = true;
  try {
    await auth.register({
      email: identifierType.value === 'email' ? values.identifier : undefined,
      phone: identifierType.value === 'phone' ? values.identifier : undefined,
      password: values.password,
      firstName: values.firstName || undefined,
    });
    await navigateTo(localePath('/'));
  } catch (error) {
    errorMessage.value =
      error instanceof ApiError ? t('login.registerError') : t('login.networkError');
  } finally {
    submitting.value = false;
  }
});

onMounted(async () => {
  if (isTma.value) {
    tmaLoading.value = true;
    const initData = getInitData();
    try {
      if (!initData) throw new Error('No initData');
      await auth.loginWithTelegramMiniApp(initData);
      await navigateTo(localePath('/'));
    } catch {
      errorMessage.value = t('login.error');
    } finally {
      tmaLoading.value = false;
    }
    return;
  }

  const botUsername = config.public.telegramBotUsername;
  if (!botUsername || !widgetContainer.value) return;

  (
    window as unknown as { onTelegramAuth: (payload: TelegramWidgetPayload) => void }
  ).onTelegramAuth = async (payload: TelegramWidgetPayload) => {
    try {
      await auth.loginWithTelegramWidget(payload);
      await navigateTo(localePath('/'));
    } catch {
      errorMessage.value = t('login.error');
    }
  };

  const script = document.createElement('script');
  script.src = 'https://telegram.org/js/telegram-widget.js?22';
  script.async = true;
  script.setAttribute('data-telegram-login', botUsername);
  script.setAttribute('data-size', 'large');
  script.setAttribute('data-onauth', 'onTelegramAuth(user)');
  script.setAttribute('data-request-access', 'write');
  script.setAttribute('data-radius', '12');
  widgetContainer.value.appendChild(script);
});
</script>

<template>
  <section class="mx-auto flex max-w-sm flex-col px-4 py-16">
    <h1 class="text-center text-2xl font-extrabold text-text">{{ $t('login.title') }}</h1>
    <p class="mt-2 text-center text-text-muted">{{ $t('login.subtitle') }}</p>

    <div v-if="!isTma" class="mt-8 flex rounded-full border border-border bg-surface p-1">
      <button
        type="button"
        class="flex-1 rounded-full py-2 text-sm font-medium transition-colors"
        :class="mode === 'login' ? 'bg-primary text-white' : 'text-text-muted'"
        @click="mode = 'login'"
      >
        {{ $t('login.tabLogin') }}
      </button>
      <button
        type="button"
        class="flex-1 rounded-full py-2 text-sm font-medium transition-colors"
        :class="mode === 'register' ? 'bg-primary text-white' : 'text-text-muted'"
        @click="mode = 'register'"
      >
        {{ $t('login.tabRegister') }}
      </button>
    </div>

    <form v-if="!isTma && mode === 'login'" class="mt-6 flex flex-col gap-4" @submit="onLogin">
      <AppInput
        v-model="loginIdentifier"
        v-bind="loginIdentifierAttrs"
        :label="$t('login.identifierLabel')"
        placeholder="you@example.com"
        :error="loginErrors.identifier"
      />
      <AppInput
        v-model="loginPassword"
        v-bind="loginPasswordAttrs"
        type="password"
        :label="$t('login.passwordLabel')"
        :error="loginErrors.password"
      />
      <AppButton type="submit" variant="primary" :disabled="submitting">
        {{ $t('login.submitLogin') }}
      </AppButton>
    </form>

    <form v-else-if="!isTma" class="mt-6 flex flex-col gap-4" @submit="onRegister">
      <div class="flex rounded-full border border-border p-1 text-sm">
        <button
          type="button"
          class="flex-1 rounded-full py-1.5 transition-colors"
          :class="identifierType === 'email' ? 'bg-bg font-medium text-text' : 'text-text-muted'"
          @click="identifierType = 'email'"
        >
          {{ $t('login.email') }}
        </button>
        <button
          type="button"
          class="flex-1 rounded-full py-1.5 transition-colors"
          :class="identifierType === 'phone' ? 'bg-bg font-medium text-text' : 'text-text-muted'"
          @click="identifierType = 'phone'"
        >
          {{ $t('login.phone') }}
        </button>
      </div>
      <AppInput
        v-model="registerIdentifier"
        v-bind="registerIdentifierAttrs"
        :type="identifierType === 'email' ? 'email' : 'tel'"
        :label="identifierType === 'email' ? $t('login.email') : $t('login.phone')"
        :placeholder="identifierType === 'email' ? 'you@example.com' : '+996 700 000 000'"
        :error="registerErrors.identifier"
      />
      <AppInput
        v-model="registerFirstName"
        v-bind="registerFirstNameAttrs"
        :label="$t('login.firstNameLabel')"
      />
      <AppInput
        v-model="registerPassword"
        v-bind="registerPasswordAttrs"
        type="password"
        :label="$t('login.passwordLabel')"
        :error="registerErrors.password"
      />
      <AppButton type="submit" variant="primary" :disabled="submitting">
        {{ $t('login.submitRegister') }}
      </AppButton>
    </form>

    <p v-if="errorMessage" class="mt-4 text-center text-sm text-primary">{{ errorMessage }}</p>

    <template v-if="isTma">
      <p v-if="tmaLoading" class="mt-8 text-center text-text-muted">{{ $t('login.loading') }}</p>
    </template>
    <template v-else>
      <div class="my-6 flex items-center gap-3 text-xs text-text-muted">
        <span class="h-px flex-1 bg-border" />
        {{ $t('login.orTelegram') }}
        <span class="h-px flex-1 bg-border" />
      </div>
      <div ref="widgetContainer" class="flex justify-center" />
    </template>
  </section>
</template>
