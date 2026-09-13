<script setup lang="ts">
const props = defineProps<{ modelValue: boolean; title?: string }>();
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();

function close(): void {
  emit('update:modelValue', false);
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
        @click.self="close"
      >
        <Transition name="modal-scale" appear>
          <section
            class="w-full max-w-md rounded-card border border-border bg-surface p-6 shadow-xl"
            role="dialog"
            aria-modal="true"
          >
            <header v-if="title" class="mb-4 flex items-center justify-between">
              <h2 class="text-lg font-semibold text-text">{{ title }}</h2>
              <button
                type="button"
                class="text-text-muted hover:text-text"
                aria-label="Close"
                @click="close"
              >
                ✕
              </button>
            </header>
            <slot />
          </section>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-scale-enter-active,
.modal-scale-leave-active {
  transition: transform 0.2s ease;
}
.modal-scale-enter-from,
.modal-scale-leave-to {
  transform: scale(0.95);
}
</style>
