import type { TrackPoint } from '@/core/api/types';

// The server accepts up to 5000 points; maps and profiles need far fewer.
const MAX_POINTS = 2000;

// Reads the track points (or route points) from a GPX file, thinned out
// evenly to at most MAX_POINTS. Browser only (DOMParser).
export function parseGpx(xml: string): TrackPoint[] {
  const doc = new DOMParser().parseFromString(xml, 'application/xml');
  if (doc.querySelector('parsererror') || doc.documentElement.localName !== 'gpx')
    throw new Error('invalid-gpx');
  let nodes = Array.from(doc.getElementsByTagNameNS('*', 'trkpt'));
  if (nodes.length === 0) nodes = Array.from(doc.getElementsByTagNameNS('*', 'rtept'));
  const points: TrackPoint[] = [];
  for (const node of nodes) {
    const rawLat = node.getAttribute('lat');
    const rawLng = node.getAttribute('lon');
    if (!rawLat?.trim() || !rawLng?.trim()) continue;
    const lat = Number(rawLat);
    const lng = Number(rawLng);
    if (!Number.isFinite(lat) || !Number.isFinite(lng) || Math.abs(lat) > 90 || Math.abs(lng) > 180)
      continue;
    const eleText = node.getElementsByTagNameNS('*', 'ele')[0]?.textContent?.trim();
    const ele = eleText ? Number(eleText) : Number.NaN;
    points.push([round(lat, 6), round(lng, 6), Number.isFinite(ele) ? Math.round(ele) : null]);
  }
  return thin(points, MAX_POINTS);
}

// At most `max` points, evenly spaced, keeping the first and the last.
export function thin(points: TrackPoint[], max: number): TrackPoint[] {
  if (points.length <= max) return points;
  const step = (points.length - 1) / (max - 1);
  return Array.from({ length: max }, (_, i) => points[Math.round(i * step)] as TrackPoint);
}

function round(n: number, digits: number): number {
  const f = 10 ** digits;
  return Math.round(n * f) / f;
}

// Distance along the track (km) at each point, for the elevation profile.
export function cumulativeKm(track: TrackPoint[]): number[] {
  if (track.length === 0) return [];
  const out = [0];
  for (let i = 1; i < track.length; i++) {
    const [lat1, lng1] = track[i - 1] as TrackPoint;
    const [lat2, lng2] = track[i] as TrackPoint;
    out.push((out[i - 1] as number) + haversine(lat1, lng1, lat2, lng2));
  }
  return out;
}

function haversine(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const toRad = (d: number): number => (d * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return 2 * 6371 * Math.asin(Math.sqrt(a));
}
