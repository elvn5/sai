import type { TrackPoint } from '~/shared/api/types';

// The server accepts up to 5000 points; maps and profiles need far fewer.
const MAX_POINTS = 2000;

// Reads the track points (or route points) from a GPX file, thinned out
// evenly to at most MAX_POINTS.
export function parseGpx(xml: string): TrackPoint[] {
  const doc = new DOMParser().parseFromString(xml, 'application/xml');
  if (doc.querySelector('parsererror')) throw new Error('Invalid GPX');
  let nodes = Array.from(doc.getElementsByTagName('trkpt'));
  if (nodes.length === 0) nodes = Array.from(doc.getElementsByTagName('rtept'));
  const points: TrackPoint[] = [];
  for (const node of nodes) {
    const lat = Number(node.getAttribute('lat'));
    const lng = Number(node.getAttribute('lon'));
    if (!Number.isFinite(lat) || !Number.isFinite(lng)) continue;
    const eleText = node.getElementsByTagName('ele')[0]?.textContent;
    const ele = eleText ? Number(eleText) : Number.NaN;
    points.push([round(lat, 6), round(lng, 6), Number.isFinite(ele) ? Math.round(ele) : null]);
  }
  if (points.length <= MAX_POINTS) return points;
  const step = (points.length - 1) / (MAX_POINTS - 1);
  return Array.from({ length: MAX_POINTS }, (_, i) => points[Math.round(i * step)] as TrackPoint);
}

function round(n: number, digits: number): number {
  const f = 10 ** digits;
  return Math.round(n * f) / f;
}

// Distance along the track (km) at each point, for the elevation profile.
export function cumulativeKm(track: TrackPoint[]): number[] {
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
