import { fromBishkekInput, nextSaturdayMorning, toBishkekInput } from '@/pages/outings/outing-form';

describe('outing times', () => {
  it('shows instants as Bishkek wall-clock time, whatever the browser zone', () => {
    expect(toBishkekInput('2026-10-17T00:30:00Z')).toBe('2026-10-17T06:30');
    expect(fromBishkekInput('2026-10-17T06:30')).toBe('2026-10-17T06:30:00+06:00');
    expect(Date.parse(fromBishkekInput(toBishkekInput('2026-10-17T00:30:00Z')))).toBe(
      Date.parse('2026-10-17T00:30:00Z'),
    );
  });

  it('suggests next Saturday at 06:00 in Bishkek', () => {
    // Thursday 15 October 2026, 12:00 in Bishkek.
    expect(nextSaturdayMorning(Date.parse('2026-10-15T06:00:00Z'))).toBe('2026-10-17T06:00');
    // On a Saturday it is the one after.
    expect(nextSaturdayMorning(Date.parse('2026-10-17T03:00:00Z'))).toBe('2026-10-24T06:00');
  });
});
