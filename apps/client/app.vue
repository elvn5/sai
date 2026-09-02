<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<style>
::view-transition-old(root),
::view-transition-new(root) {
  animation: none;
  mix-blend-mode: normal;
}
::view-transition-old(root) {
  z-index: 1;
}
::view-transition-new(root) {
  z-index: 9999;
}

/* Disable the default color mode transition to prevent blinking */
html.color-mode-transition,
html.color-mode-transition *,
html.color-mode-transition *:before,
html.color-mode-transition *:after {
  transition: none !important;
}
</style>
