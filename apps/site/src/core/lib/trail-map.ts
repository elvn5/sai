import type { Trail } from '@/core/api/types';

export function trailStart(
  trail: Pick<Trail, 'startLatitude' | 'startLongitude'>,
): [number, number] | null {
  const lat = trail.startLatitude;
  const lng = trail.startLongitude;
  return lat !== null &&
    lng !== null &&
    Number.isFinite(lat) &&
    Number.isFinite(lng) &&
    lat >= -90 &&
    lat <= 90 &&
    lng >= -180 &&
    lng <= 180
    ? [lat, lng]
    : null;
}
