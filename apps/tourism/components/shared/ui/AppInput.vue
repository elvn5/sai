<script setup lang="ts">
defineOptions({ inheritAttrs: false });

defineProps<{
  modelValue?: string;
  label?: string;
  type?: string;
  placeholder?: string;
  error?: string;
}>();

defineEmits<{ 'update:modelValue': [value: string] }>();
</script>

<template>
  <label class="flex flex-col gap-1.5 text-sm">
    <span v-if="label" class="font-medium text-text">{{ label }}</span>
    <input
      :value="modelValue ?? ''"
      :type="type ?? 'text'"
      :placeholder="placeholder"
      v-bind="$attrs"
      class="rounded-xl border border-border bg-surface px-4 py-2.5 text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
      :class="{ 'border-primary': error }"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <span v-if="error" class="text-xs text-primary">{{ error }}</span>
  </label>
</template>
