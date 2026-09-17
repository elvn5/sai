<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import { ApiError } from '~/shared/api/client';
import type { TelegramWidgetPayload } from '~/shared/api/types';
import type { PendingVerification } from '~/shared/lib/auth';

const { t } = useI18n();
const localePath = useLocalePath();
const { isTma, getInitData } = useTelegramWebApp();
const auth = useAuth();
const config = useRuntimeConfig();
const colorMode = useColorMode();

const mode = ref<'login' | 'register' | 'verify'>('login');
const identifierType = ref<'email' | 'phone'>('email');
const errorMessage = ref<string | null>(null);
const submitting = ref(false);
const tmaLoading = ref(false);
const widgetContainer = ref<HTMLDivElement | null>(null);
const telegramUnavailable = ref(false);

const pendingVerification = ref<PendingVerification | null>(null);
const pendingPassword = ref('');
const resendCooldown = ref(0);
let resendTimer: ReturnType<typeof setInterval> | undefined;
let verifyPollTimer: ReturnType<typeof setInterval> | undefined;

const telegramDeepLink = computed(() => {
  const verification = pendingVerification.value;
  const botUsername = config.public.telegramBotUsername;
  if (!verification?.token || !botUsername) return null;
  return `https://t.me/${botUsername}?start=verify_${verification.token}`;
});

useSeoMeta({
  title: () => t('login.title'),
  description: () => t('login.subtitle'),
  robots: 'noindex',
});

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

const verifySchema = toTypedSchema(
  z.object({
    code: z.string().regex(/^\d{6}$/, t('login.verify.invalidFormat')),
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

const {
  handleSubmit: handleVerifySubmit,
  defineField: defineVerifyField,
  errors: verifyErrors,
  resetForm: resetVerifyForm,
} = useForm({ validationSchema: verifySchema });
const [verifyCodeValue, verifyCodeAttrs] = defineVerifyField('code');

function startResendCooldown(): void {
  resendCooldown.value = 60;
  clearInterval(resendTimer);
  resendTimer = setInterval(() => {
    resendCooldown.value -= 1;
    if (resendCooldown.value <= 0) clearInterval(resendTimer);
  }, 1000);
}

function stopVerificationPoll(): void {
  clearInterval(verifyPollTimer);
  verifyPollTimer = undefined;
}

async function completeTelegramLogin(): Promise<void> {
  if (!pendingVerification.value) return;
  try {
    const result = await auth.login({
      identifier: pendingVerification.value.identifier,
      password: pendingPassword.value,
    });
    if (!result) {
      await navigateTo(localePath('/'));
    }
  } catch {
    errorMessage.value = t('login.networkError');
  }
}

function startVerificationPoll(): void {
  stopVerificationPoll();
  verifyPollTimer = setInterval(async () => {
    if (!pendingVerification.value) return;
    try {
      const verified = await auth.checkVerificationStatus(pendingVerification.value.identifier);
      if (verified) {
        stopVerificationPoll();
        await completeTelegramLogin();
      }
    } catch {
      // Transient network hiccup — keep polling silently.
    }
  }, 3000);
}

function enterVerifyMode(verification: PendingVerification): void {
  pendingVerification.value = verification;
  errorMessage.value = null;
  resetVerifyForm();
  mode.value = 'verify';
  startResendCooldown();
  if (verification.method === 'telegram') {
    startVerificationPoll();
  }
}

const onLogin = handleLoginSubmit(async (values) => {
  errorMessage.value = null;
  submitting.value = true;
  try {
    const verification = await auth.login({
      identifier: values.identifier,
      password: values.password,
    });
    if (verification) {
      pendingPassword.value = values.password;
      enterVerifyMode(verification);
      return;
    }
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
    const verification = await auth.register({
      email: identifierType.value === 'email' ? values.identifier : undefined,
      phone: identifierType.value === 'phone' ? values.identifier : undefined,
      password: values.password,
      firstName: values.firstName || undefined,
    });
    pendingPassword.value = values.password;
    enterVerifyMode(verification);
  } catch (error) {
    errorMessage.value =
      error instanceof ApiError && error.status !== 500 && error.status !== 503
        ? t('login.registerError')
        : t('login.verify.sendError');
  } finally {
    submitting.value = false;
  }
});

const onVerify = handleVerifySubmit(async (values) => {
  if (!pendingVerification.value) return;
  errorMessage.value = null;
  submitting.value = true;
  try {
    await auth.verifyCode({ identifier: pendingVerification.value.identifier, code: values.code });
    await navigateTo(localePath('/'));
  } catch (error) {
    errorMessage.value =
      error instanceof ApiError ? t('login.verify.invalidCode') : t('login.networkError');
  } finally {
    submitting.value = false;
  }
});

async function handleResend(): Promise<void> {
  if (!pendingVerification.value || resendCooldown.value > 0) return;
  errorMessage.value = null;
  try {
    pendingVerification.value = await auth.resendCode(pendingVerification.value.identifier);
    startResendCooldown();
  } catch {
    errorMessage.value = t('login.verify.resendError');
  }
}

function backToLogin(): void {
  mode.value = 'login';
  pendingVerification.value = null;
  pendingPassword.value = '';
  errorMessage.value = null;
  clearInterval(resendTimer);
  stopVerificationPoll();
}

function mountTelegramWidget(): void {
  const botUsername = config.public.telegramBotUsername;
  if (!botUsername || !widgetContainer.value) {
    telegramUnavailable.value = true;
    return;
  }

  widgetContainer.value.replaceChildren();

  const script = document.createElement('script');
  script.src = 'https://telegram.org/js/telegram-widget.js?22';
  script.async = true;
  script.setAttribute('data-telegram-login', botUsername);
  script.setAttribute('data-size', 'large');
  script.setAttribute('data-onauth', 'onTelegramAuth(user)');
  script.setAttribute('data-request-access', 'write');
  script.setAttribute('data-radius', '12');
  // Telegram's "logged in as X" chip renders with an opaque dark backdrop we
  // can't restyle (cross-origin iframe) — dropping the avatar/personalization
  // in light mode keeps it to the plain button, which never shows that backdrop.
  script.setAttribute('data-userpic', colorMode.value === 'light' ? 'false' : 'true');
  widgetContainer.value.appendChild(script);
}

watch(
  () => colorMode.value,
  () => {
    if (!isTma.value && !telegramUnavailable.value) mountTelegramWidget();
  },
);

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

  mountTelegramWidget();
});

onBeforeUnmount(() => {
  clearInterval(resendTimer);
  stopVerificationPoll();
});
</script>

<template>
  <section class="mx-auto flex max-w-sm flex-col px-4 py-16">
    <template v-if="mode === 'verify'">
      <h1 class="text-center text-2xl font-extrabold text-text">{{ $t('login.verify.title') }}</h1>

      <template v-if="pendingVerification?.method === 'email'">
        <p class="mt-2 text-center text-text-muted">
          {{ $t('login.verify.subtitleEmail', { identifier: pendingVerification?.identifier }) }}
        </p>

        <form class="mt-8 flex flex-col gap-4" @submit="onVerify">
          <UFormGroup :label="$t('login.verify.codeLabel')" :error="verifyErrors.code">
            <UInput
              v-model="verifyCodeValue"
              v-bind="verifyCodeAttrs"
              inputmode="numeric"
              maxlength="6"
              placeholder="000000"
            />
          </UFormGroup>
          <UButton type="submit" color="primary" block :loading="submitting">
            {{ $t('login.verify.submit') }}
          </UButton>
        </form>

        <button
          type="button"
          class="mt-4 text-center text-sm text-primary-600 disabled:text-text-muted dark:text-primary-400"
          :disabled="resendCooldown > 0"
          @click="handleResend"
        >
          {{
            resendCooldown > 0
              ? $t('login.verify.resendCooldown', { seconds: resendCooldown })
              : $t('login.verify.resend')
          }}
        </button>
      </template>

      <template v-else-if="pendingVerification?.method === 'telegram'">
        <p class="mt-2 text-center text-text-muted">{{ $t('login.verify.subtitleTelegram') }}</p>

        <UButton
          class="mt-8"
          color="primary"
          block
          :to="telegramDeepLink ?? undefined"
          :disabled="!telegramDeepLink"
          target="_blank"
        >
          {{ $t('login.verify.openTelegram') }}
        </UButton>

        <div class="mt-4 flex items-center justify-center gap-2 text-sm text-text-muted">
          <UIcon name="i-heroicons-arrow-path" class="h-4 w-4 animate-spin" />
          {{ $t('login.verify.waitingTelegram') }}
        </div>

        <button
          type="button"
          class="mt-4 text-center text-sm text-primary-600 disabled:text-text-muted dark:text-primary-400"
          :disabled="resendCooldown > 0"
          @click="handleResend"
        >
          {{
            resendCooldown > 0
              ? $t('login.verify.resendLinkCooldown', { seconds: resendCooldown })
              : $t('login.verify.resendLink')
          }}
        </button>
      </template>

      <button
        type="button"
        class="mt-2 text-center text-sm text-text-muted hover:underline"
        @click="backToLogin"
      >
        {{ $t('login.verify.back') }}
      </button>
    </template>

    <template v-else>
      <h1 class="text-center text-2xl font-extrabold text-text">{{ $t('login.title') }}</h1>
      <p class="mt-2 text-center text-text-muted">{{ $t('login.subtitle') }}</p>

      <div v-if="!isTma" class="mt-8 flex rounded-full border border-border bg-surface p-1">
        <button
          type="button"
          class="flex-1 rounded-full py-2 text-sm font-medium transition-colors"
          :class="mode === 'login' ? 'bg-primary-600 text-white dark:bg-primary-500' : 'text-text-muted'"
          @click="mode = 'login'"
        >
          {{ $t('login.tabLogin') }}
        </button>
        <button
          type="button"
          class="flex-1 rounded-full py-2 text-sm font-medium transition-colors"
          :class="mode === 'register' ? 'bg-primary-600 text-white dark:bg-primary-500' : 'text-text-muted'"
          @click="mode = 'register'"
        >
          {{ $t('login.tabRegister') }}
        </button>
      </div>

      <form v-if="!isTma && mode === 'login'" class="mt-6 flex flex-col gap-4" @submit="onLogin">
        <UFormGroup :label="$t('login.identifierLabel')" :error="loginErrors.identifier">
          <UInput v-model="loginIdentifier" v-bind="loginIdentifierAttrs" placeholder="you@example.com" />
        </UFormGroup>
        <UFormGroup :label="$t('login.passwordLabel')" :error="loginErrors.password">
          <UInput v-model="loginPassword" v-bind="loginPasswordAttrs" type="password" />
        </UFormGroup>
        <UButton type="submit" color="primary" block :loading="submitting">
          {{ $t('login.submitLogin') }}
        </UButton>
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
        <p class="text-xs text-text-muted">
          {{
            identifierType === 'email'
              ? $t('login.verify.hintEmail')
              : $t('login.verify.hintTelegram')
          }}
        </p>
        <UFormGroup
          :label="identifierType === 'email' ? $t('login.email') : $t('login.phone')"
          :error="registerErrors.identifier"
        >
          <UInput
            v-model="registerIdentifier"
            v-bind="registerIdentifierAttrs"
            :type="identifierType === 'email' ? 'email' : 'tel'"
            :placeholder="identifierType === 'email' ? 'you@example.com' : '+996 700 000 000'"
          />
        </UFormGroup>
        <UFormGroup :label="$t('login.firstNameLabel')">
          <UInput v-model="registerFirstName" v-bind="registerFirstNameAttrs" />
        </UFormGroup>
        <UFormGroup :label="$t('login.passwordLabel')" :error="registerErrors.password">
          <UInput v-model="registerPassword" v-bind="registerPasswordAttrs" type="password" />
        </UFormGroup>
        <UButton type="submit" color="primary" block :loading="submitting">
          {{ $t('login.submitRegister') }}
        </UButton>
      </form>
    </template>

    <p v-if="errorMessage" class="mt-4 text-center text-sm text-primary-600 dark:text-primary-400">
      {{ errorMessage }}
    </p>

    <template v-if="mode !== 'verify'">
      <template v-if="isTma">
        <p v-if="tmaLoading" class="mt-8 text-center text-text-muted">{{ $t('login.loading') }}</p>
      </template>
      <template v-else>
        <div class="my-6 flex items-center gap-3 text-xs text-text-muted">
          <span class="h-px flex-1 bg-border" />
          {{ $t('login.orTelegram') }}
          <span class="h-px flex-1 bg-border" />
        </div>
        <p v-if="telegramUnavailable" class="text-center text-sm text-text-muted">
          {{ $t('login.telegramUnavailable') }}
        </p>
        <div v-else class="flex min-h-[56px] justify-center py-2">
          <div ref="widgetContainer" class="telegram-widget-frame" />
        </div>
      </template>
    </template>
  </section>
</template>
