import { trailStart } from '@/core/lib/trail-map';

describe('trail map locations', () => {
  it('keeps valid zero coordinates and excludes absent or invalid locations', () => {
    expect(trailStart({ startLatitude: 0, startLongitude: 0 })).toEqual([0, 0]);
    expect(trailStart({ startLatitude: 42.5, startLongitude: 74.5 })).toEqual([42.5, 74.5]);
    for (const startLatitude of [null, Number.NaN, Number.POSITIVE_INFINITY, 91, -91]) {
      expect(trailStart({ startLatitude, startLongitude: 74 })).toBeNull();
    }
    expect(trailStart({ startLatitude: 42, startLongitude: null })).toBeNull();
    expect(trailStart({ startLatitude: 42, startLongitude: 181 })).toBeNull();
  });
});
