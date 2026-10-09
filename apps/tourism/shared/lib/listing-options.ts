import { CancellationPolicy, Destination, HotelType } from '~/shared/api/types';

// Mirrors services/listings/internal/models/options.go — keep the two in sync.

export interface ListingOption {
  value: string;
  // i18n key for the label.
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
  { value: HotelType.HOTEL, labelKey: 'hotels.types.hotel', icon: 'i-heroicons-building-office-2' },
  { value: HotelType.GUESTHOUSE, labelKey: 'hotels.types.guesthouse', icon: 'i-heroicons-home' },
  { value: HotelType.HOSTEL, labelKey: 'hotels.types.hostel', icon: 'i-heroicons-user-group' },
  { value: HotelType.YURT_CAMP, labelKey: 'hotels.types.yurt_camp', icon: 'i-heroicons-fire' },
  { value: HotelType.RESORT, labelKey: 'hotels.types.resort', icon: 'i-heroicons-sun' },
  {
    value: HotelType.APARTMENT,
    labelKey: 'hotels.types.apartment',
    icon: 'i-heroicons-building-office',
  },
];

export const HOTEL_AMENITY_OPTIONS: ListingOption[] = [
  { value: 'wifi', labelKey: 'listing.amenities.wifi', icon: 'i-heroicons-wifi' },
  { value: 'breakfast', labelKey: 'listing.amenities.breakfast', icon: 'i-heroicons-cake' },
  { value: 'parking', labelKey: 'listing.amenities.parking', icon: 'i-heroicons-truck' },
  {
    value: 'airport_transfer',
    labelKey: 'listing.amenities.airport_transfer',
    icon: 'i-heroicons-paper-airplane',
  },
  { value: 'hot_water', labelKey: 'listing.amenities.hot_water', icon: 'i-heroicons-beaker' },
  { value: 'heating', labelKey: 'listing.amenities.heating', icon: 'i-heroicons-fire' },
  { value: 'kitchen', labelKey: 'listing.amenities.kitchen', icon: 'i-heroicons-squares-2x2' },
  {
    value: 'restaurant',
    labelKey: 'listing.amenities.restaurant',
    icon: 'i-heroicons-building-storefront',
  },
  { value: 'sauna', labelKey: 'listing.amenities.sauna', icon: 'i-heroicons-cloud' },
  { value: 'pool', labelKey: 'listing.amenities.pool', icon: 'i-heroicons-lifebuoy' },
  { value: 'beach', labelKey: 'listing.amenities.beach', icon: 'i-heroicons-sun' },
  { value: 'laundry', labelKey: 'listing.amenities.laundry', icon: 'i-heroicons-arrow-path' },
  {
    value: 'family_friendly',
    labelKey: 'listing.amenities.family_friendly',
    icon: 'i-heroicons-heart',
  },
  {
    value: 'pets_allowed',
    labelKey: 'listing.amenities.pets_allowed',
    icon: 'i-heroicons-face-smile',
  },
  { value: 'tours_desk', labelKey: 'listing.amenities.tours_desk', icon: 'i-heroicons-map' },
  { value: 'horse_riding', labelKey: 'listing.amenities.horse_riding', icon: 'i-heroicons-flag' },
];

export const CANCELLATION_OPTIONS: ListingOption[] = Object.values(CancellationPolicy).map(
  (value) => ({ value, labelKey: `listing.cancellation.${value}` }),
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
  { value: 'trekking', labelKey: 'localGuides.specializations.trekking', icon: 'i-heroicons-map' },
  {
    value: 'horse_riding',
    labelKey: 'localGuides.specializations.horse_riding',
    icon: 'i-heroicons-flag',
  },
  {
    value: 'mountaineering',
    labelKey: 'localGuides.specializations.mountaineering',
    icon: 'i-heroicons-arrow-trending-up',
  },
  {
    value: 'culture',
    labelKey: 'localGuides.specializations.culture',
    icon: 'i-heroicons-building-library',
  },
  {
    value: 'nomad_life',
    labelKey: 'localGuides.specializations.nomad_life',
    icon: 'i-heroicons-fire',
  },
  {
    value: 'city',
    labelKey: 'localGuides.specializations.city',
    icon: 'i-heroicons-building-office-2',
  },
  { value: 'photo', labelKey: 'localGuides.specializations.photo', icon: 'i-heroicons-camera' },
  { value: 'ski', labelKey: 'localGuides.specializations.ski', icon: 'i-heroicons-sparkles' },
  { value: 'jeep', labelKey: 'localGuides.specializations.jeep', icon: 'i-heroicons-truck' },
  { value: 'food', labelKey: 'localGuides.specializations.food', icon: 'i-heroicons-cake' },
];

export const MIN_RATING_OPTIONS = [4.5, 4, 3.5, 3];

export function findOption(options: ListingOption[], value: string): ListingOption | undefined {
  return options.find((option) => option.value === value);
}
