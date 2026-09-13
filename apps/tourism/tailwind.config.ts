import typography from '@tailwindcss/typography';

export default {
  darkMode: 'class',
  plugins: [typography],
  theme: {
    extend: {
      colors: {
        // Named "brand", not "primary" — @nuxt/ui's module unconditionally
        // overwrites a color literally named "primary" (and "gray") with an
        // internal CSS-variable placeholder. app.config.ts points
        // ui.primary at this scale by name instead.
        brand: {
          50: '#fef2f2',
          100: '#fde2e2',
          200: '#fbc5c5',
          300: '#f59d9d',
          400: '#e63946',
          500: '#d62828',
          600: '#b91f1f',
          700: '#991b1b',
          800: '#7f1d1d',
          900: '#641414',
          950: '#3d0a0a',
        },
        secondary: {
          50: '#fffaeb',
          100: '#fff3c4',
          200: '#ffe58a',
          300: '#ffd350',
          400: '#ffb703',
          500: '#f4a261',
          600: '#e08c46',
          700: '#c26f2e',
          800: '#9c5720',
          900: '#7a4419',
          950: '#452509',
        },
        bg: 'var(--color-bg)',
        surface: 'var(--color-surface)',
        border: 'var(--color-border)',
        text: 'var(--color-text)',
        'text-muted': 'var(--color-text-muted)',
      },
      borderRadius: {
        card: '1rem',
      },
    },
  },
};
