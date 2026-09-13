<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost';
    disabled?: boolean;
    type?: 'button' | 'submit';
  }>(),
  { variant: 'primary', disabled: false, type: 'button' },
);

const emit = defineEmits<{ click: [event: MouseEvent] }>();
const { isTma, haptic } = useTelegramWebApp();

const variantClasses: Record<string, string> = {
  primary: 'bg-primary text-white hover:bg-primary-hover',
  secondary: 'bg-secondary text-[#1a1a1a] hover:bg-secondary-hover',
  ghost: 'bg-transparent text-text border border-border hover:bg-surface',
};

function onClick(event: MouseEvent): void {
  if (props.disabled) return;
  if (isTma.value) haptic('light');
  emit('click', event);
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    class="inline-flex items-center justify-center gap-2 rounded-full px-5 py-2.5 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
    :class="variantClasses[variant]"
    @click="onClick"
  >
    <slot />
  </button>
</template>
