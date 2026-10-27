import {
  type Terrain,
  decodeTerrarium,
  drapedTrack,
  surfaceHeight,
  terrainBounds,
  terrainPosition,
} from '@/core/lib/terrain';

const terrain: Terrain = {
  south: 42,
  north: 42.01,
  west: 74,
  east: 74.01,
  columns: 1,
  rows: 1,
  heights: new Float32Array([1000, 1100, 1200, 1500]),
  base: 1000,
  scale: 0.01,
};

describe('terrain surface', () => {
  it('decodes positive, negative and fractional elevations', () => {
    expect(decodeTerrarium(137, 219, 68)).toBe(2523.265625);
    expect(decodeTerrarium(128, 0, 0)).toBe(0);
    expect(decodeTerrarium(127, 255, 0)).toBe(-1);
  });
  it('pads even a coincident track to show nearby mountains', () => {
    const bounds = terrainBounds([
      [42, 74, null],
      [42, 74, null],
    ]);
    expect(bounds.north).toBeGreaterThan(42.01);
    expect(bounds.west).toBeLessThan(73.99);
  });
  it('samples both rendered triangles exactly, including the shared diagonal', () => {
    expect(surfaceHeight(terrain, 42.0075, 74.0025)).toBeCloseTo(1075);
    expect(surfaceHeight(terrain, 42.0025, 74.0075)).toBeCloseTo(1325);
    expect(surfaceHeight(terrain, 42.005, 74.005)).toBeCloseTo(1150);
    expect(surfaceHeight(terrain, 42, 74.01)).toBeCloseTo(1500);
  });
  it('drapes sparse GPX over the terrain without changing source elevations', () => {
    const track: [number, number, number | null][] = [
      [42.01, 74, 9999],
      [42, 74.01, null],
    ];
    const points = drapedTrack(terrain, track);
    expect(points.length).toBeGreaterThan(2);
    expect(points[0]?.[1]).toBeCloseTo(0.025);
    expect(points.at(-1)?.[1]).toBeCloseTo(5.025);
    expect(track[0]?.[2]).toBe(9999);
  });
  it('uses one metric scale for mountains and route', () => {
    const a = terrainPosition(terrain, 42, 74, 1000);
    const b = terrainPosition(terrain, 42.001, 74, 1100);
    expect((b[1] - a[1]) / Math.abs(b[2] - a[2])).toBeCloseTo(100 / 111.32);
  });
});
