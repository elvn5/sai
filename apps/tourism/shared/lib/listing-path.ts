import { type ListingRef, ReviewTargetType } from '~/shared/api/types';

// The public page of one of a partner's listings, before localePath: tours
// live under their own locale.
export function listingPath(listing: ListingRef): string {
  switch (listing.targetType) {
    case ReviewTargetType.HOTEL:
      return `/hotels/${listing.slug}`;
    case ReviewTargetType.LOCAL_GUIDE:
      return `/local-guides/${listing.slug}`;
    case ReviewTargetType.TOUR: {
      const prefix = listing.locale && listing.locale !== 'ru' ? `/${listing.locale}` : '';
      return `${prefix}/tours/${listing.slug}`;
    }
  }
}
