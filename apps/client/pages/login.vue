<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div>
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Sign in to your account
        </h2>
      </div>
      <UForm :state="state" class="mt-8 space-y-6" @submit="onSubmit">
        <UFormGroup label="Email address" name="email">
          <UInput v-model="state.email" type="email" autocomplete="email" required />
        </UFormGroup>

        <UFormGroup label="Password" name="password">
          <UInput v-model="state.password" type="password" autocomplete="current-password" required />
        </UFormGroup>

        <div>
          <UButton type="submit" color="primary" block>
            Sign in
          </UButton>
        </div>
      </UForm>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'

const state = reactive({
  email: '',
  password: ''
})

const onSubmit = () => {
  console.log('Submitted', state)
}
</script>
