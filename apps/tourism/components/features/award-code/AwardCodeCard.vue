<script setup lang="ts">
import { encode } from 'uqr';

// The user's personal award code, as text and as a QR code linking to the
// partner's award page with the code filled in.
const props = defineProps<{ code: string }>();

const localePath = useLocalePath();
const requestUrl = useRequestURL();
const copied = ref(false);

const formatted = computed(() => `${props.code.slice(0, 4)}-${props.code.slice(4)}`);
const awardUrl = computed(
  () =>
    `${requestUrl.origin}${localePath({ path: '/achievements/award', query: { code: props.code } })}`,
);
const qr = computed(() => encode(awardUrl.value, { border: 1 }));

async function copyCode(): Promise<void> {
  try {
    await navigator.clipboard.writeText(props.code);
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    // Clipboard blocked: the code stays visible to copy by hand.
  }
}
</script>

<template>
  <div class="code-card relative overflow-hidden rounded-card border border-border bg-surface p-5">
    <div class="flex flex-col items-center gap-4">
      <svg
        :viewBox="`0 0 ${qr.size} ${qr.size}`"
        class="h-36 w-36 shrink-0 rounded-lg bg-white p-1"
        shape-rendering="crispEdges"
        role="img"
        :aria-label="$t('hallOfFame.code.qrLabel')"
      >
        <template v-for="(row, y) in qr.data" :key="y">
          <rect v-for="(on, x) in row" v-show="on" :key="x" :x="x" :y="y" width="1" height="1" fill="#0f172a" />
        </template>
      </svg>
      <div class="flex flex-col items-center gap-2 text-center">
        <p class="text-sm font-medium text-text-muted">{{ $t('hallOfFame.code.title') }}</p>
        <button
          type="button"
          class="whitespace-nowrap font-mono text-2xl font-extrabold tracking-[0.15em] text-text transition hover:text-brand-600"
          :aria-label="$t('hallOfFame.code.copy')"
          @click="copyCode"
        >
          {{ formatted }}
        </button>
        <p class="text-xs text-text-muted">
          {{ copied ? $t('hallOfFame.code.copied') : $t('hallOfFame.code.hint') }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.code-card::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(120deg, transparent 30%, rgb(var(--color-brand-500) / 0.6), transparent 70%);
  background-size: 250% 100%;
  animation: sweep 4s linear infinite;
  -webkit-mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}
@keyframes sweep {
  to {
    background-position: -250% 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .code-card::before {
    animation: none;
  }
}
</style>
