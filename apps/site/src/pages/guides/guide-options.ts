import { GuideDifficulty, GuideSeason } from '@/core/api/types';

// The fixed equipment vocabulary of guides (labels under
// guides.equipmentItems.*). Older guides may hold free-text entries: they
// show without an icon.
export const EQUIPMENT_CATALOG = [
  { key: 'tent', icon: 'icon-[mdi--tent]' },
  { key: 'backpack', icon: 'icon-[mdi--bag-personal]' },
  { key: 'boots', icon: 'icon-[mdi--shoe-sneaker]' },
  { key: 'water', icon: 'icon-[mdi--water]' },
  { key: 'flashlight', icon: 'icon-[mdi--flashlight]' },
  { key: 'firstAid', icon: 'icon-[mdi--medical-bag]' },
  { key: 'compass', icon: 'icon-[mdi--compass-outline]' },
  { key: 'rain', icon: 'icon-[mdi--umbrella-outline]' },
  { key: 'sunglasses', icon: 'icon-[mdi--sunglasses]' },
  { key: 'stove', icon: 'icon-[mdi--fire]' },
  { key: 'rope', icon: 'icon-[mdi--carabiner]' },
  { key: 'gps', icon: 'icon-[mdi--crosshairs-gps]' },
] as const;

export const DIFFICULTY_ICONS: Record<GuideDifficulty, string> = {
  [GuideDifficulty.EASY]: 'icon-[mdi--signal-cellular-1]',
  [GuideDifficulty.MODERATE]: 'icon-[mdi--signal-cellular-2]',
  [GuideDifficulty.HARD]: 'icon-[mdi--signal-cellular-3]',
};

export const SEASON_ICONS: Record<GuideSeason, string> = {
  [GuideSeason.SUMMER]: 'icon-[mdi--white-balance-sunny]',
  [GuideSeason.WINTER]: 'icon-[mdi--snowflake]',
  [GuideSeason.ALL_YEAR]: 'icon-[mdi--calendar-sync]',
};

export function equipmentIcon(entry: string): string | null {
  return EQUIPMENT_CATALOG.find((item) => item.key === entry)?.icon ?? null;
}

// Translation key of a catalog entry; free text is shown as is.
export function equipmentLabelKey(entry: string): string | null {
  return EQUIPMENT_CATALOG.some((item) => item.key === entry)
    ? `guides.equipmentItems.${entry}`
    : null;
}
