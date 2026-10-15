import type { TrackPoint } from '@/core/api/types';
import { cumulativeKm, parseGpx, thin } from '@/core/lib/gpx';

const GPX = `<?xml version="1.0"?>
<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk><trkseg>
    <trkpt lat="42.5641234" lon="74.4822"><ele>2140.4</ele></trkpt>
    <trkpt lat="42.5700" lon="74.4900"></trkpt>
    <trkpt lat="bad" lon="74.5"><ele>2300</ele></trkpt>
    <trkpt lat="42.5800" lon="74.5000"><ele>2500</ele></trkpt>
  </trkseg></trk>
</gpx>`;

describe('parseGpx', () => {
  it('reads track points, rounding them and skipping broken ones', () => {
    expect(parseGpx(GPX)).toEqual([
      [42.564123, 74.4822, 2140],
      [42.57, 74.49, null],
      [42.58, 74.5, 2500],
    ]);
  });

  it('falls back to route points and rejects what is not XML', () => {
    const route = '<gpx><rte><rtept lat="1" lon="2"><ele>3</ele></rtept></rte></gpx>';
    expect(parseGpx(route)).toEqual([[1, 2, 3]]);
    expect(() => parseGpx('<gpx><trk>')).toThrow();
  });
});

describe('thin', () => {
  it('keeps the first and last points, evenly spaced', () => {
    const points = Array.from({ length: 101 }, (_, i): TrackPoint => [i, 0, null]);
    const kept = thin(points, 11);
    expect(kept).toHaveLength(11);
    expect(kept.map((p) => p[0])).toEqual([0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100]);
    expect(thin(points.slice(0, 5), 11)).toHaveLength(5);
  });
});

describe('cumulativeKm', () => {
  it('adds up great-circle distances along the track', () => {
    // A degree of latitude is about 111.2 km.
    const km = cumulativeKm([
      [42, 74, null],
      [43, 74, null],
      [44, 74, null],
    ]);
    expect(km[0]).toBe(0);
    expect(km[1]).toBeCloseTo(111.2, 0);
    expect(km[2]).toBeCloseTo(222.4, 0);
  });
});
