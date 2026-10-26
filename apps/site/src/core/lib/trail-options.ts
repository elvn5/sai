// Mirrors services/trails/internal/models/options.go — keep the two in
// sync. Labels are translation keys under trails.*; icons
// are static CSS classes (written out in full so Tailwind generates them).

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
  { value: 'hike', icon: 'icon-[mdi--hiking]' },
  { value: 'trek', icon: 'icon-[mdi--tent]' },
  { value: 'climb', icon: 'icon-[mdi--image-filter-hdr]' },
  { value: 'ski_tour', icon: 'icon-[mdi--ski]' },
  { value: 'ice_climb', icon: 'icon-[mdi--snowflake]' },
  { value: 'rock_climb', icon: 'icon-[mdi--carabiner]' },
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
  { value: 'spring', icon: 'icon-[mdi--flower-tulip-outline]' },
  { value: 'summer', icon: 'icon-[mdi--white-balance-sunny]' },
  { value: 'autumn', icon: 'icon-[mdi--leaf-maple]' },
  { value: 'winter', icon: 'icon-[mdi--snowflake]' },
];

export const GEAR: TrailOption[] = [
  { value: 'helmet', icon: 'icon-[mdi--racing-helmet]' },
  { value: 'crampons', icon: 'icon-[mdi--shoe-cleat]' },
  { value: 'microspikes', icon: 'icon-[mdi--shoe-sneaker]' },
  { value: 'ice_axe', icon: 'icon-[mdi--pickaxe]' },
  { value: 'harness', icon: 'icon-[mdi--carabiner]' },
  { value: 'rope', icon: 'icon-[mdi--jump-rope]' },
  { value: 'ice_screws', icon: 'icon-[mdi--screw-lag]' },
  { value: 'trekking_poles', icon: 'icon-[mdi--hiking]' },
  { value: 'headlamp', icon: 'icon-[mdi--flashlight]' },
  { value: 'tent', icon: 'icon-[mdi--tent]' },
  { value: 'sleeping_bag', icon: 'icon-[mdi--sleep]' },
  { value: 'stove', icon: 'icon-[mdi--fire]' },
  { value: 'avalanche_kit', icon: 'icon-[mdi--shovel]' },
  { value: 'snowshoes', icon: 'icon-[mdi--snowshoeing]' },
  { value: 'gaiters', icon: 'icon-[mdi--shoe-formal]' },
  { value: 'sunglasses', icon: 'icon-[mdi--sunglasses]' },
];

export const TRAIL_STATES: TrailOption[] = [
  { value: 'clear', icon: 'icon-[mdi--check-circle-outline]' },
  { value: 'wet', icon: 'icon-[mdi--water-outline]' },
  { value: 'patchy_snow', icon: 'icon-[mdi--snowflake-melt]' },
  { value: 'snow', icon: 'icon-[mdi--snowflake]' },
  { value: 'deep_snow', icon: 'icon-[mdi--snowflake-alert]' },
  { value: 'ice', icon: 'icon-[mdi--snowflake-variant]' },
];

export const TRACTIONS: TrailOption[] = [
  { value: 'none', icon: 'icon-[mdi--shoe-sneaker]' },
  { value: 'microspikes', icon: 'icon-[mdi--shoe-cleat]' },
  { value: 'crampons', icon: 'icon-[mdi--pickaxe]' },
];

export const AVALANCHE_RISKS = ['low', 'moderate', 'considerable', 'high'] as const;

export function optionIcon(options: TrailOption[], value: string): string {
  return (
    options.find((option) => option.value === value)?.icon ?? 'icon-[mdi--help-circle-outline]'
  );
}

// Reports older than this are shown as stale.
export const FRESH_REPORT_DAYS = 14;

// Whole days from a Bishkek calendar date (YYYY-MM-DD) to `now`.
export function daysSince(isoDate: string, now: number = Date.now()): number {
  const then = new Date(`${isoDate}T00:00:00+06:00`).getTime();
  return Math.max(0, Math.floor((now - then) / 86_400_000));
}

// Grade badge colors, easiest to hardest.
export function gradeClass(grade: string): string {
  if (grade === 'trail') return 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-200';
  if (grade.startsWith('1')) return 'bg-sky-500/15 text-sky-800 dark:text-sky-200';
  if (grade.startsWith('2')) return 'bg-amber-500/15 text-amber-800 dark:text-amber-200';
  if (grade.startsWith('3')) return 'bg-orange-500/15 text-orange-800 dark:text-orange-200';
  return 'bg-red-500/15 text-red-800 dark:text-red-200';
}

// "1b" → "1Б": grades are written with Cyrillic letters.
export function formatGrade(grade: string): string {
  return grade === 'trail' ? grade : grade.replace('a', 'А').replace('b', 'Б');
}
