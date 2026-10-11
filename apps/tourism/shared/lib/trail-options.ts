// Mirrors services/trails/internal/models/options.go — keep the two in sync.
// Labels are i18n keys under `trails.*`.

export interface TrailOption {
  value: string;
  icon: string;
}

export const AREAS = [
  'ala_archa',
  'alamedin',
  'chunkurchak',
  'issyk_ata',
  'kegeti',
  'sokuluk',
  'chon_kemin',
  'karakol',
  'jeti_oguz',
  'suusamyr',
  'song_kol',
  'kel_suu',
  'alay',
  'central_tien_shan',
  'other',
] as const;

export const KINDS: TrailOption[] = [
  { value: 'hike', icon: 'i-mdi-hiking' },
  { value: 'trek', icon: 'i-mdi-tent' },
  { value: 'climb', icon: 'i-mdi-image-filter-hdr' },
  { value: 'ski_tour', icon: 'i-mdi-ski' },
  { value: 'ice_climb', icon: 'i-mdi-snowflake' },
  { value: 'rock_climb', icon: 'i-mdi-carabiner' },
];

// Russian mountaineering grades; "trail" is a walk without one.
export const GRADES = [
  'trail',
  '1a',
  '1b',
  '2a',
  '2b',
  '3a',
  '3b',
  '4a',
  '4b',
  '5a',
  '5b',
  '6a',
  '6b',
] as const;

export const SEASONS: TrailOption[] = [
  { value: 'spring', icon: 'i-mdi-flower-tulip-outline' },
  { value: 'summer', icon: 'i-mdi-white-balance-sunny' },
  { value: 'autumn', icon: 'i-mdi-leaf-maple' },
  { value: 'winter', icon: 'i-mdi-snowflake' },
];

export const GEAR: TrailOption[] = [
  { value: 'helmet', icon: 'i-mdi-racing-helmet' },
  { value: 'crampons', icon: 'i-mdi-shoe-cleat' },
  { value: 'microspikes', icon: 'i-mdi-shoe-sneaker' },
  { value: 'ice_axe', icon: 'i-mdi-pickaxe' },
  { value: 'harness', icon: 'i-mdi-carabiner' },
  { value: 'rope', icon: 'i-mdi-jump-rope' },
  { value: 'ice_screws', icon: 'i-mdi-screw-lag' },
  { value: 'trekking_poles', icon: 'i-mdi-hiking' },
  { value: 'headlamp', icon: 'i-mdi-flashlight' },
  { value: 'tent', icon: 'i-mdi-tent' },
  { value: 'sleeping_bag', icon: 'i-mdi-sleep' },
  { value: 'stove', icon: 'i-mdi-fire' },
  { value: 'avalanche_kit', icon: 'i-mdi-shovel' },
  { value: 'snowshoes', icon: 'i-mdi-snowshoeing' },
  { value: 'gaiters', icon: 'i-mdi-shoe-formal' },
  { value: 'sunglasses', icon: 'i-mdi-sunglasses' },
];

export const TRAIL_STATES: TrailOption[] = [
  { value: 'clear', icon: 'i-mdi-check-circle-outline' },
  { value: 'wet', icon: 'i-mdi-water-outline' },
  { value: 'patchy_snow', icon: 'i-mdi-snowflake-melt' },
  { value: 'snow', icon: 'i-mdi-snowflake' },
  { value: 'deep_snow', icon: 'i-mdi-snowflake-alert' },
  { value: 'ice', icon: 'i-mdi-snowflake-variant' },
];

export const TRACTIONS: TrailOption[] = [
  { value: 'none', icon: 'i-mdi-shoe-sneaker' },
  { value: 'microspikes', icon: 'i-mdi-shoe-cleat' },
  { value: 'crampons', icon: 'i-mdi-pickaxe' },
];

export const AVALANCHE_RISKS = ['low', 'moderate', 'considerable', 'high'] as const;

export const TRANSPORTS: TrailOption[] = [
  { value: 'car', icon: 'i-mdi-car' },
  { value: 'bus', icon: 'i-mdi-bus' },
  { value: 'arranged', icon: 'i-mdi-van-passenger' },
];

export const LEVELS: TrailOption[] = [
  { value: 'beginner', icon: 'i-mdi-signal-cellular-1' },
  { value: 'intermediate', icon: 'i-mdi-signal-cellular-2' },
  { value: 'advanced', icon: 'i-mdi-signal-cellular-3' },
];

export function optionIcon(options: TrailOption[], value: string): string {
  return options.find((o) => o.value === value)?.icon ?? 'i-mdi-help-circle-outline';
}

// Reports older than this are shown as stale.
export const FRESH_REPORT_DAYS = 14;

export function daysSince(isoDate: string): number {
  const then = new Date(`${isoDate}T00:00:00+06:00`).getTime();
  return Math.max(0, Math.floor((Date.now() - then) / 86_400_000));
}

// Grade badge colors, easiest to hardest.
export function gradeClass(grade: string): string {
  if (grade === 'trail') return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300';
  if (grade.startsWith('1')) return 'bg-sky-500/15 text-sky-700 dark:text-sky-300';
  if (grade.startsWith('2')) return 'bg-amber-500/15 text-amber-700 dark:text-amber-300';
  if (grade.startsWith('3')) return 'bg-orange-500/15 text-orange-700 dark:text-orange-300';
  return 'bg-red-500/15 text-red-700 dark:text-red-300';
}

// "1b" → "1Б": grades are written with Cyrillic letters.
export function formatGrade(grade: string): string {
  return grade === 'trail' ? grade : grade.replace('a', 'А').replace('b', 'Б');
}
