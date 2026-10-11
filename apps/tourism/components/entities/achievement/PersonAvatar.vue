<script setup lang="ts">
import type { Person } from '~/shared/api/types';

const props = withDefaults(defineProps<{ person: Person; size?: string }>(), { size: 'h-10 w-10' });
const name = computed(() => personName(props.person));
</script>

<template>
  <img v-if="person.avatarUrl" :src="person.avatarUrl" :alt="name" class="shrink-0 rounded-full object-cover" :class="size" />
  <span
    v-else
    class="flex shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-secondary-500 font-bold text-white"
    :class="size"
  >
    {{ name.replace('@', '').charAt(0).toUpperCase() }}
  </span>
</template>
