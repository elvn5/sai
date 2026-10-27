import { satelliteExtent, satelliteUV } from '@/core/lib/satellite';

describe('satellite georeferencing', () => {
  it('maps geographic north to the top and east to the right without mirroring', () => {
    const atlas = { zoom: 1, left: 1, top: 0, columns: 1, rows: 1 };
    expect(satelliteUV(atlas, 0, 0)).toEqual([0, 0]);
    expect(satelliteUV(atlas, 0, 180)).toEqual([1, 0]);
    const north = satelliteUV(atlas, 85.0511287798066, 0);
    expect(north[0]).toBe(0);
    expect(north[1]).toBeCloseTo(1);
  });
  it('covers terrain bounds with a bounded atlas for both short and long routes', () => {
    for (const bounds of [
      { north: 42.6, south: 42.5, west: 74.4, east: 74.6 },
      { north: 42.6, south: 42.1, west: 78.2, east: 78.8 },
    ]) {
      const atlas = satelliteExtent(bounds);
      expect(atlas.columns * atlas.rows).toBeLessThanOrEqual(64);
      expect(Math.max(atlas.columns, atlas.rows)).toBeLessThanOrEqual(12);
      for (const [lat, lon] of [
        [bounds.north, bounds.west],
        [bounds.south, bounds.east],
      ]) {
        for (const value of satelliteUV(atlas, lat ?? 0, lon ?? 0)) {
          expect(value).toBeGreaterThanOrEqual(0);
          expect(value).toBeLessThanOrEqual(1);
        }
      }
    }
  });
});
