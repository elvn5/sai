import { type Terrain, tilePoint } from '@/core/lib/terrain';

export interface SatelliteExtent {
  zoom: number;
  left: number;
  top: number;
  columns: number;
  rows: number;
}
export interface SatelliteAtlas extends SatelliteExtent {
  canvas: HTMLCanvasElement;
}

// At most 64 visible tiles, six concurrent requests, no offline tile archive.
// The atlas is only assembled in memory for this 3D view.
export function satelliteExtent(
  bounds: Pick<Terrain, 'north' | 'south' | 'east' | 'west'>,
): SatelliteExtent {
  for (let zoom = 15; zoom >= 0; zoom--) {
    const [x0, y0] = tilePoint(bounds.north, bounds.west, zoom);
    const [x1, y1] = tilePoint(bounds.south, bounds.east, zoom);
    const left = Math.floor(x0);
    const top = Math.floor(y0);
    const columns = Math.floor(x1) - left + 1;
    const rows = Math.floor(y1) - top + 1;
    if (columns * rows <= 64 && Math.max(columns, rows) <= 12)
      return { zoom, left, top, columns, rows };
  }
  throw new Error('Unsupported imagery extent');
}

// Web Mercator UVs align the satellite pixels with the geographic mesh.
// V is inverted because CanvasTexture flips its image on upload.
export function satelliteUV(atlas: SatelliteExtent, lat: number, lon: number): [number, number] {
  const [x, y] = tilePoint(lat, lon, atlas.zoom);
  return [(x - atlas.left) / atlas.columns, 1 - (y - atlas.top) / atlas.rows];
}

export async function loadSatellite(
  terrain: Terrain,
  signal: AbortSignal,
): Promise<SatelliteAtlas> {
  const extent = satelliteExtent(terrain);
  const canvas = document.createElement('canvas');
  canvas.width = extent.columns * 256;
  canvas.height = extent.rows * 256;
  const context = canvas.getContext('2d');
  if (!context) throw new Error('No imagery decoder');
  let next = 0;
  const worker = async (): Promise<void> => {
    while (next < extent.columns * extent.rows) {
      signal.throwIfAborted();
      const index = next++;
      const col = index % extent.columns;
      const row = Math.floor(index / extent.columns);
      const response = await fetch(
        `https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/${extent.zoom}/${extent.top + row}/${extent.left + col}`,
        { signal, credentials: 'omit' },
      );
      if (!response.ok || !response.headers.get('content-type')?.startsWith('image/'))
        throw new Error('Imagery unavailable');
      const bitmap = await createImageBitmap(await response.blob());
      try {
        signal.throwIfAborted();
        context.drawImage(bitmap, col * 256, row * 256, 256, 256);
      } finally {
        bitmap.close();
      }
    }
  };
  await Promise.all(Array.from({ length: 6 }, worker));
  signal.throwIfAborted();
  return { ...extent, canvas };
}
