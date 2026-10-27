import type { Request, Response } from 'express';

const cache = new Map<string, Buffer>();
const pending = new Map<string, Promise<Buffer>>();
const ORIGIN = 'https://s3.amazonaws.com/elevation-tiles-prod/terrarium';

export function terrainTileKey(z: string, x: string, y: string): string | null {
  if (![z, x, y].every((value) => /^\d{1,5}$/.test(value))) return null;
  const zoom = Number(z);
  if (zoom > 12 || Number(x) >= 2 ** zoom || Number(y) >= 2 ** zoom) return null;
  return `${zoom}/${Number(x)}/${Number(y)}`;
}

// Fixed upstream, bounded memory/concurrency, no arbitrary proxy URLs.
export async function serveTerrain(req: Request, res: Response): Promise<void> {
  const key = terrainTileKey(
    String(req.params['z']),
    String(req.params['x']),
    String(req.params['y']),
  );
  if (!key) {
    res.status(400).end();
    return;
  }
  try {
    let data = cache.get(key);
    if (!data) {
      let job = pending.get(key);
      if (!job) {
        if (pending.size >= 32) {
          res.status(503).setHeader('Retry-After', '5').end();
          return;
        }
        job = (async () => {
          const response = await fetch(`${ORIGIN}/${key}.png`, {
            signal: AbortSignal.timeout(15000),
            redirect: 'error',
          });
          if (!response.ok || !response.headers.get('content-type')?.startsWith('image/png'))
            throw new Error('Terrain upstream failed');
          const bytes = Buffer.from(await response.arrayBuffer());
          if (bytes.length > 512000 || bytes.readUInt32BE(0) !== 0x89504e47)
            throw new Error('Invalid tile');
          if (cache.size >= 128) {
            const first = cache.keys().next().value;
            if (first) cache.delete(first);
          }
          cache.set(key, bytes);
          return bytes;
        })().finally(() => pending.delete(key));
        pending.set(key, job);
      }
      data = await job;
    }
    res.type('image/png').setHeader('Cache-Control', 'public, max-age=604800').send(data);
  } catch {
    res.status(503).setHeader('Cache-Control', 'no-store').setHeader('Retry-After', '30').end();
  }
}
