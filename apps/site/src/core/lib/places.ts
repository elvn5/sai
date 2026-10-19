import { isPlatformBrowser } from '@angular/common';
import { type HttpResourceRef, httpResource } from '@angular/common/http';
import { PLATFORM_ID, inject } from '@angular/core';

// Places held per tour id: paid bookings and those awaiting payment.
export type TakenPlaces = Record<string, number>;

// Below this many free places a tour shows "N places left".
export const FEW_PLACES = 5;

// How many places bookings hold on the given tours (services/payments).
// It changes by the minute, so it's asked in the browser only and never
// cached: the server-rendered page stays the same for everyone.
export function placesResource(
  ids: () => string[] | undefined,
): HttpResourceRef<TakenPlaces | undefined> {
  const browser = isPlatformBrowser(inject(PLATFORM_ID));
  return httpResource<TakenPlaces>(() => {
    const list = ids();
    if (!browser || !list?.length) return undefined;
    return { url: '/orders/places', params: { ids: list.join(',') }, transferCache: false };
  });
}

// Free places on a tour, or null when it has no limit (or before the count
// is known).
export function placesLeft(
  tour: { id: string; capacity: number | null },
  taken: TakenPlaces | undefined,
): number | null {
  if (tour.capacity == null || !taken) return null;
  return Math.max(0, tour.capacity - (taken[tour.id] ?? 0));
}
