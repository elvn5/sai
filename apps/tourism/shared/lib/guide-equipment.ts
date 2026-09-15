// Fixed equipment vocabulary users pick from when creating a guide (icons +
// i18n keys under guides.equipmentItems.*). Guides created before this
// existed may have arbitrary free-text entries — those just render without
// an icon (see guides/[slug].vue).
export const EQUIPMENT_CATALOG = [
  { key: 'tent', icon: 'i-mdi-tent' },
  { key: 'backpack', icon: 'i-mdi-bag-personal' },
  { key: 'boots', icon: 'i-mdi-shoe-sneaker' },
  { key: 'water', icon: 'i-mdi-water' },
  { key: 'flashlight', icon: 'i-mdi-flashlight' },
  { key: 'firstAid', icon: 'i-mdi-medical-bag' },
  { key: 'compass', icon: 'i-mdi-compass-outline' },
  { key: 'rain', icon: 'i-mdi-umbrella-outline' },
  { key: 'sunglasses', icon: 'i-mdi-sunglasses' },
  { key: 'stove', icon: 'i-mdi-fire' },
  { key: 'rope', icon: 'i-mdi-carabiner' },
  { key: 'gps', icon: 'i-mdi-crosshairs-gps' },
] as const;

export type EquipmentKey = (typeof EQUIPMENT_CATALOG)[number]['key'];
