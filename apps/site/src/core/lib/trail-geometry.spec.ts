import { projectTrack, steepPoints, validMarkers } from '@/core/lib/trail-geometry';

describe('route geometry', () => {
  it('preserves metric elevation ratios and keeps missing heights finite', () => {
    const points = projectTrack([
      [42, 74, 2000],
      [42.001, 74, 2050],
      [42.002, 74, null],
    ]);
    expect(
      (points[1]?.[1] ?? Number.NaN) /
        Math.abs((points[1]?.[2] ?? Number.NaN) - (points[0]?.[2] ?? Number.NaN)),
    ).toBeCloseTo(50 / 111.32, 3);
    expect(points.flat().every(Number.isFinite)).toBe(true);
    expect(points[2]?.[1]).toBe(0);
  });
  it('handles empty and coincident tracks', () => {
    expect(projectTrack([])).toEqual([]);
    expect(
      projectTrack([
        [42, 74, null],
        [42, 74, null],
      ]),
    ).toEqual([
      [0, 0, 0],
      [0, 0, 0],
    ]);
  });
  it('suggests sustained climbs and descents, ignoring short jitter and missing elevation', () => {
    expect(
      steepPoints([
        [42, 74, 2000],
        [42.001, 74, 2030],
        [42.002, 74, 2000],
      ]),
    ).toEqual([1, 2]);
    expect(
      steepPoints([
        [42, 74, 2000],
        [42.00001, 74, 2010],
        [42.001, 74, 2001],
      ]),
    ).toEqual([]);
    expect(
      steepPoints([
        [42, 74, null],
        [42.001, 74, 2030],
      ]),
    ).toEqual([]);
  });
  it('drops invalid marker references', () => {
    expect(
      validMarkers(
        [
          { pointIndex: 0, label: 'A' },
          { pointIndex: 1, label: 'B' },
        ],
        [[42, 74, 1000]],
      ),
    ).toEqual([{ pointIndex: 0, label: 'A' }]);
  });
});
