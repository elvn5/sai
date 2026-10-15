import { CancellationPolicy, Destination, HotelType } from '@/core/api/types';

// Mirrors services/listings/internal/models/options.go — keep the two in
// sync. Icons are static CSS classes (written out in full so Tailwind
// generates them).

export interface ListingOption {
  value: string;
  // Translation key of the label.
  labelKey: string;
  icon?: string;
}

export const DESTINATION_OPTIONS: ListingOption[] = Object.values(Destination).map((value) => ({
  value,
  labelKey: `destinations.${value}`,
}));

// The places most visitors come for, shown on the home page. Issyk-Kul alone
// draws about half of all of Kyrgyzstan's tourist traffic.
export const FEATURED_DESTINATIONS: Destination[] = [
  Destination.ISSYK_KUL,
  Destination.KARAKOL,
  Destination.SONG_KOL,
  Destination.ALA_ARCHA,
  Destination.BISHKEK,
  Destination.OSH,
];

export const HOTEL_TYPE_OPTIONS: ListingOption[] = [
  {
    value: HotelType.HOTEL,
    labelKey: 'hotels.types.hotel',
    icon: 'icon-[heroicons--building-office-2]',
  },
  {
    value: HotelType.GUESTHOUSE,
    labelKey: 'hotels.types.guesthouse',
    icon: 'icon-[heroicons--home]',
  },
  {
    value: HotelType.HOSTEL,
    labelKey: 'hotels.types.hostel',
    icon: 'icon-[heroicons--user-group]',
  },
  {
    value: HotelType.YURT_CAMP,
    labelKey: 'hotels.types.yurt_camp',
    icon: 'icon-[heroicons--fire]',
  },
  { value: HotelType.RESORT, labelKey: 'hotels.types.resort', icon: 'icon-[heroicons--sun]' },
  {
    value: HotelType.APARTMENT,
    labelKey: 'hotels.types.apartment',
    icon: 'icon-[heroicons--building-office]',
  },
];

export const HOTEL_AMENITY_OPTIONS: ListingOption[] = [
  { value: 'wifi', labelKey: 'listing.amenities.wifi', icon: 'icon-[heroicons--wifi]' },
  { value: 'breakfast', labelKey: 'listing.amenities.breakfast', icon: 'icon-[heroicons--cake]' },
  { value: 'parking', labelKey: 'listing.amenities.parking', icon: 'icon-[heroicons--truck]' },
  {
    value: 'airport_transfer',
    labelKey: 'listing.amenities.airport_transfer',
    icon: 'icon-[heroicons--paper-airplane]',
  },
  { value: 'hot_water', labelKey: 'listing.amenities.hot_water', icon: 'icon-[heroicons--beaker]' },
  { value: 'heating', labelKey: 'listing.amenities.heating', icon: 'icon-[heroicons--fire]' },
  {
    value: 'kitchen',
    labelKey: 'listing.amenities.kitchen',
    icon: 'icon-[heroicons--squares-2x2]',
  },
  {
    value: 'restaurant',
    labelKey: 'listing.amenities.restaurant',
    icon: 'icon-[heroicons--building-storefront]',
  },
  { value: 'sauna', labelKey: 'listing.amenities.sauna', icon: 'icon-[heroicons--cloud]' },
  { value: 'pool', labelKey: 'listing.amenities.pool', icon: 'icon-[heroicons--lifebuoy]' },
  { value: 'beach', labelKey: 'listing.amenities.beach', icon: 'icon-[heroicons--sun]' },
  { value: 'laundry', labelKey: 'listing.amenities.laundry', icon: 'icon-[heroicons--arrow-path]' },
  {
    value: 'family_friendly',
    labelKey: 'listing.amenities.family_friendly',
    icon: 'icon-[heroicons--heart]',
  },
  {
    value: 'pets_allowed',
    labelKey: 'listing.amenities.pets_allowed',
    icon: 'icon-[heroicons--face-smile]',
  },
  { value: 'tours_desk', labelKey: 'listing.amenities.tours_desk', icon: 'icon-[heroicons--map]' },
  {
    value: 'horse_riding',
    labelKey: 'listing.amenities.horse_riding',
    icon: 'icon-[heroicons--flag]',
  },
];

export const CANCELLATION_OPTIONS: ListingOption[] = Object.values(CancellationPolicy).map(
  (value) => ({
    value,
    labelKey: `listing.cancellation.${value}`,
  }),
);

export const GUIDE_LANGUAGE_OPTIONS: ListingOption[] = [
  'ru',
  'ky',
  'en',
  'de',
  'fr',
  'es',
  'it',
  'zh',
  'ja',
  'ko',
  'tr',
  'ar',
  'kk',
  'uz',
].map((value) => ({ value, labelKey: `localGuides.languages.${value}` }));

export const GUIDE_SPECIALIZATION_OPTIONS: ListingOption[] = [
  {
    value: 'trekking',
    labelKey: 'localGuides.specializations.trekking',
    icon: 'icon-[heroicons--map]',
  },
  {
    value: 'horse_riding',
    labelKey: 'localGuides.specializations.horse_riding',
    icon: 'icon-[heroicons--flag]',
  },
  {
    value: 'mountaineering',
    labelKey: 'localGuides.specializations.mountaineering',
    icon: 'icon-[heroicons--arrow-trending-up]',
  },
  {
    value: 'culture',
    labelKey: 'localGuides.specializations.culture',
    icon: 'icon-[heroicons--building-library]',
  },
  {
    value: 'nomad_life',
    labelKey: 'localGuides.specializations.nomad_life',
    icon: 'icon-[heroicons--fire]',
  },
  {
    value: 'city',
    labelKey: 'localGuides.specializations.city',
    icon: 'icon-[heroicons--building-office-2]',
  },
  {
    value: 'photo',
    labelKey: 'localGuides.specializations.photo',
    icon: 'icon-[heroicons--camera]',
  },
  { value: 'ski', labelKey: 'localGuides.specializations.ski', icon: 'icon-[heroicons--sparkles]' },
  { value: 'jeep', labelKey: 'localGuides.specializations.jeep', icon: 'icon-[heroicons--truck]' },
  { value: 'food', labelKey: 'localGuides.specializations.food', icon: 'icon-[heroicons--cake]' },
];

export const MIN_RATING_OPTIONS = [4.5, 4, 3.5, 3];

export function findOption(options: ListingOption[], value: string): ListingOption | undefined {
  return options.find((option) => option.value === value);
}

// The options for `values`, in the given order, skipping unknown ones.
export function pickOptions(options: ListingOption[], values: string[]): ListingOption[] {
  return values.map((value) => findOption(options, value)).filter((option) => option !== undefined);
}
