<script setup lang="ts">
const props = defineProps<{
  modelValue: string | string[] | undefined;
  options: { value: string; icon: string; label: string }[];
  multiple?: boolean;
}>();

const emit = defineEmits<{ 'update:modelValue': [string | string[]] }>();

function isActive(value: string): boolean {
  return props.multiple
    ? Array.isArray(props.modelValue) && props.modelValue.includes(value)
    : props.modelValue === value;
}

function toggle(value: string): void {
  if (!props.multiple) {
    emit('update:modelValue', value);
    return;
  }
  const current = Array.isArray(props.modelValue) ? props.modelValue : [];
  emit(
    'update:modelValue',
    current.includes(value) ? current.filter((entry) => entry !== value) : [...current, value],
  );
}
</script>

<template>
  <div class="flex flex-wrap gap-3">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="flex w-20 flex-col items-center gap-1.5 rounded-card border px-3 py-3 text-xs font-medium transition-colors"
      :class="
        isActive(option.value)
          ? 'icon-select-active border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
          : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
      "
      :aria-pressed="isActive(option.value)"
      @click="toggle(option.value)"
    >
      <UIcon :name="option.icon" class="h-6 w-6" />
      <span class="text-center leading-tight">{{ option.label }}</span>
    </button>
  </div>
</template>
