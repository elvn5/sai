import { onMounted, readonly, ref } from 'vue';

export type ColorTheme = 'red' | 'azure';

const STORAGE_KEY = 'color-theme';
const theme = ref<ColorTheme>('red');
let initialized = false;

function applyTheme(value: ColorTheme): void {
  if (typeof document === 'undefined') return;
  if (value === 'azure') {
    document.documentElement.setAttribute('data-color-theme', 'azure');
  } else {
    document.documentElement.removeAttribute('data-color-theme');
  }
}

export function useColorTheme() {
  onMounted(() => {
    if (initialized) return;
    initialized = true;

    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'red' || stored === 'azure') {
      theme.value = stored;
    }
    applyTheme(theme.value);
  });

  function setTheme(value: ColorTheme): void {
    theme.value = value;
    applyTheme(value);
    localStorage.setItem(STORAGE_KEY, value);
  }

  function toggleTheme(): void {
    setTheme(theme.value === 'red' ? 'azure' : 'red');
  }

  return {
    theme: readonly(theme),
    setTheme,
    toggleTheme,
  };
}
