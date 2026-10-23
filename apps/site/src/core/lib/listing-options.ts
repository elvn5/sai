import { CancellationPolicy, Destination } from '@/core/api/types';

// Mirrors services/listings/internal/models/options.go — keep the two in
// sync (hotel and local guide options are left out while those sections
// are paused). Icons are static CSS classes (written out in full so Tailwind
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

export const CANCELLATION_OPTIONS: ListingOption[] = Object.values(CancellationPolicy).map(
  (value) => ({
    value,
    labelKey: `listing.cancellation.${value}`,
  }),
);

export function findOption(options: ListingOption[], value: string): ListingOption | undefined {
  return options.find((option) => option.value === value);
}

// The options for `values`, in the given order, skipping unknown ones.
export function pickOptions(options: ListingOption[], values: string[]): ListingOption[] {
  return values.map((value) => findOption(options, value)).filter((option) => option !== undefined);
}
