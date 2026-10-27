import type { TrackPoint } from '@/core/api/types';

export interface Terrain {
  south: number;
  north: number;
  west: number;
  east: number;
  columns: number;
  rows: number;
  heights: Float32Array;
  base: number;
  scale: number;
}

export function terrainBounds(
  track: TrackPoint[],
): Pick<Terrain, 'south' | 'north' | 'west' | 'east'> {
  if (!track.length) throw new Error('Empty track');
  const latitudes = track.map((p) => p[0]);
  const longitudes = track.map((p) => p[1]);
  const south = Math.min(...latitudes);
  const north = Math.max(...latitudes);
  const west = Math.min(...longitudes);
  const east = Math.max(...longitudes);
  if (south < -80 || north > 80 || east - west > 180) throw new Error('Unsupported extent');
  const cos = Math.cos(((south + north) * Math.PI) / 360);
  const padding = Math.max(
    1200,
    (north - south) * 111320 * 0.18,
    (east - west) * 111320 * cos * 0.18,
  );
  return {
    south: south - padding / 111320,
    north: north + padding / 111320,
    west: west - padding / (111320 * cos),
    east: east + padding / (111320 * cos),
  };
}

export function tilePoint(lat: number, lon: number, zoom: number): [number, number] {
  const radians = (lat * Math.PI) / 180;
  return [
    ((lon + 180) / 360) * 2 ** zoom,
    ((1 - Math.asinh(Math.tan(radians)) / Math.PI) / 2) * 2 ** zoom,
  ];
}

export function decodeTerrarium(red: number, green: number, blue: number): number {
  return red * 256 + green + blue / 256 - 32768;
}

// Browser-only, called after WebGL support is established. The server caches
// fixed-provider tiles; no API key or third-party cookies reach the browser.
export async function loadTerrain(track: TrackPoint[], signal: AbortSignal): Promise<Terrain> {
  const bounds = terrainBounds(track);
  let zoom = 12;
  const extent = (): number[] => {
    const [left, top] = tilePoint(bounds.north, bounds.west, zoom);
    const [right, bottom] = tilePoint(bounds.south, bounds.east, zoom);
    return [Math.floor(left), Math.floor(top), Math.floor(right), Math.floor(bottom)];
  };
  let [left = 0, top = 0, right = 0, bottom = 0] = extent();
  while ((right - left + 1) * (bottom - top + 1) > 16 && zoom > 0) {
    zoom--;
    [left = 0, top = 0, right = 0, bottom = 0] = extent();
  }
  const tiles = new Map<string, Uint8ClampedArray>();
  const jobs: Promise<void>[] = [];
  for (let y = top; y <= bottom; y++) {
    for (let x = left; x <= right; x++) {
      jobs.push(
        (async () => {
          const response = await fetch(`/terrain/${zoom}/${x}/${y}.png`, { signal });
          if (!response.ok) throw new Error('Terrain unavailable');
          const bitmap = await createImageBitmap(await response.blob(), {
            colorSpaceConversion: 'none',
          });
          try {
            const canvas = document.createElement('canvas');
            canvas.width = canvas.height = 256;
            const context = canvas.getContext('2d', { willReadFrequently: true });
            if (!context) throw new Error('No terrain decoder');
            context.drawImage(bitmap, 0, 0);
            tiles.set(`${x}/${y}`, context.getImageData(0, 0, 256, 256).data);
          } finally {
            bitmap.close();
          }
        })(),
      );
    }
  }
  await Promise.all(jobs);
  signal.throwIfAborted();
  const width =
    (bounds.east - bounds.west) *
    111320 *
    Math.cos(((bounds.north + bounds.south) * Math.PI) / 360);
  const depth = (bounds.north - bounds.south) * 111320;
  const columns = Math.max(32, Math.round((192 * width) / Math.max(width, depth)));
  const rows = Math.max(32, Math.round((192 * depth) / Math.max(width, depth)));
  const heights = new Float32Array((columns + 1) * (rows + 1));
  const pixelHeight = (px: number, py: number): number => {
    const x = Math.max(left * 256, Math.min((right + 1) * 256 - 1, px));
    const y = Math.max(top * 256, Math.min((bottom + 1) * 256 - 1, py));
    const data = tiles.get(`${Math.floor(x / 256)}/${Math.floor(y / 256)}`);
    if (!data) throw new Error('Missing terrain tile');
    const i = ((y % 256) * 256 + (x % 256)) * 4;
    return decodeTerrarium(data[i] ?? 0, data[i + 1] ?? 0, data[i + 2] ?? 0);
  };
  for (let row = 0; row <= rows; row++) {
    for (let col = 0; col <= columns; col++) {
      const [tx, ty] = tilePoint(
        bounds.north - (row / rows) * (bounds.north - bounds.south),
        bounds.west + (col / columns) * (bounds.east - bounds.west),
        zoom,
      );
      const px = tx * 256;
      const py = ty * 256;
      const x = Math.floor(px);
      const y = Math.floor(py);
      const u = px - x;
      const v = py - y;
      heights[row * (columns + 1) + col] =
        pixelHeight(x, y) * (1 - u) * (1 - v) +
        pixelHeight(x + 1, y) * u * (1 - v) +
        pixelHeight(x, y + 1) * (1 - u) * v +
        pixelHeight(x + 1, y + 1) * u * v;
    }
  }
  return {
    ...bounds,
    columns,
    rows,
    heights,
    base: Math.min(...heights),
    scale: 12 / Math.max(width, depth),
  };
}

// Match the mesh's triangle diagonal, rather than bilinear interpolation:
// a route on a saddle must sit on the rendered surface too.
export function surfaceHeight(t: Terrain, lat: number, lon: number): number {
  const x = Math.max(0, Math.min(t.columns, ((lon - t.west) / (t.east - t.west)) * t.columns));
  const y = Math.max(0, Math.min(t.rows, ((t.north - lat) / (t.north - t.south)) * t.rows));
  const col = Math.min(t.columns - 1, Math.floor(x));
  const row = Math.min(t.rows - 1, Math.floor(y));
  const u = x - col;
  const v = y - row;
  const i = row * (t.columns + 1) + col;
  const a = t.heights[i] ?? 0;
  const b = t.heights[i + 1] ?? 0;
  const c = t.heights[i + t.columns + 1] ?? 0;
  const d = t.heights[i + t.columns + 2] ?? 0;
  return u + v <= 1 ? a + (b - a) * u + (c - a) * v : d + (c - d) * (1 - u) + (b - d) * (1 - v);
}

export function terrainPosition(
  t: Terrain,
  lat: number,
  lon: number,
  height: number,
): [number, number, number] {
  return [
    (lon - (t.west + t.east) / 2) *
      111320 *
      Math.cos(((t.north + t.south) * Math.PI) / 360) *
      t.scale,
    (height - t.base) * t.scale,
    ((t.north + t.south) / 2 - lat) * 111320 * t.scale,
  ];
}

export function drapedTrack(t: Terrain, track: TrackPoint[]): [number, number, number][] {
  const result: [number, number, number][] = [];
  for (let i = 0; i < track.length; i++) {
    const point = track[i];
    if (!point) continue;
    const previous = track[i - 1] ?? point;
    const steps = Math.min(
      Math.max(2, Math.floor(24000 / track.length)),
      Math.max(
        1,
        Math.ceil(
          Math.max(
            (Math.abs(point[0] - previous[0]) / (t.north - t.south)) * t.rows,
            (Math.abs(point[1] - previous[1]) / (t.east - t.west)) * t.columns,
          ) * 4,
        ),
      ),
    );
    for (let step = 1; step <= steps; step++) {
      const lat = previous[0] + ((point[0] - previous[0]) * step) / steps;
      const lon = previous[1] + ((point[1] - previous[1]) * step) / steps;
      const p = terrainPosition(t, lat, lon, surfaceHeight(t, lat, lon));
      p[1] += 0.025;
      result.push(p);
    }
  }
  return result;
}
