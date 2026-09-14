import typography from '@tailwindcss/typography';

const SHADES = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950] as const;

// @nuxt/ui's own generated `primary-*`/`secondary-*` utilities expect their
// CSS variables to hold a raw "R G B" triplet, wrapped as `rgb(var(--x) /
// <alpha>)`. Matching that convention here (instead of a plain hex var) keeps
// our own `brand-*`/`secondary-*` classes and Nuxt UI's classes resolving to
// the same runtime-swappable color (see assets/css/tokens.css), and adds
// `/opacity` modifier support for free.
function withOpacity(variableName: string) {
  return ({ opacityValue }: { opacityValue?: string }) => {
    if (opacityValue !== undefined) {
      return `rgb(var(${variableName}) / ${opacityValue})`;
    }
    return `rgb(var(${variableName}))`;
  };
}

function colorScale(prefix: string) {
  return Object.fromEntries(
    SHADES.map((shade) => [shade, withOpacity(`--color-${prefix}-${shade}`)]),
  );
}

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
        // Values read from CSS custom properties (see assets/css/tokens.css)
        // so the color theme (default red vs. [data-color-theme="azure"])
        // can be swapped at runtime without a rebuild.
        brand: colorScale('brand'),
        secondary: colorScale('secondary'),
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
