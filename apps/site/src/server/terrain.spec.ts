import { terrainTileKey } from '@/server/terrain';

describe('terrain proxy coordinates', () => {
  it('permits only bounded integer tile coordinates', () => {
    expect(terrainTileKey('12', '2895', '1512')).toBe('12/2895/1512');
    expect(terrainTileKey('0', '0', '0')).toBe('0/0/0');
    for (const values of [
      ['13', '0', '0'],
      ['12', '4096', '0'],
      ['1', '-1', '0'],
      ['2', '0', '../secret'],
      ['2', '1.5', '0'],
    ]) {
      expect(terrainTileKey(values[0] ?? '', values[1] ?? '', values[2] ?? '')).toBeNull();
    }
  });
});
