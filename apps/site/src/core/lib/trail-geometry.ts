import type { TrackPoint, TrailMarker } from '@/core/api/types';
import { cumulativeKm } from '@/core/lib/gpx';

// Local metric projection: north is -Z, elevation is Y. One shared scale
// preserves the actual slope rather than inventing surrounding terrain.
export function projectTrack(track: TrackPoint[]): [number, number, number][] {
  const first = track[0];
  if (!first) return [];
  const [lat, lng] = first;
  const heights = track.flatMap((p) => (p[2] === null ? [] : [p[2]]));
  const base = heights.length ? Math.min(...heights) : 0;
  const raw = track.map((p): [number, number, number] => [
    (p[1] - lng) * 111320 * Math.cos((lat * Math.PI) / 180),
    (p[2] ?? base) - base,
    -(p[0] - lat) * 111320,
  ]);
  const xs = raw.map((p) => p[0]);
  const zs = raw.map((p) => p[2]);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minZ = Math.min(...zs);
  const maxZ = Math.max(...zs);
  const scale = 10 / Math.max(maxX - minX, maxZ - minZ, ...raw.map((p) => p[1]), 1);
  return raw.map(([x, y, z]) => [
    (x - (minX + maxX) / 2) * scale,
    y * scale,
    (z - (minZ + maxZ) / 2) * scale,
  ]);
}

// Suggest sustained gradients over >=100 m, not GPS jitter between samples.
export function steepPoints(track: TrackPoint[]): number[] {
  const distances = cumulativeKm(track);
  const result: number[] = [];
  let start = 0;
  for (let end = 1; end < track.length; end++) {
    const endHeight = track[end]?.[2];
    const startHeight = track[start]?.[2];
    if (endHeight == null || startHeight == null) {
      start = end;
      continue;
    }
    const meters = ((distances[end] ?? 0) - (distances[start] ?? 0)) * 1000;
    if (meters < 100) continue;
    if (Math.abs(endHeight - startHeight) / meters >= 0.2) result.push(end);
    start = end;
  }
  return result.slice(0, 100);
}

export function validMarkers(markers: TrailMarker[], track: TrackPoint[]): TrailMarker[] {
  return markers.filter(
    (m) => Number.isInteger(m.pointIndex) && m.pointIndex >= 0 && m.pointIndex < track.length,
  );
}
