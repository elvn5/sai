import { KINDS, daysSince, formatGrade, gradeClass, optionIcon } from '@/core/lib/trail-options';

describe('trail options', () => {
  it('writes grades with Cyrillic letters', () => {
    expect(formatGrade('1b')).toBe('1Б');
    expect(formatGrade('3a')).toBe('3А');
    expect(formatGrade('trail')).toBe('trail');
  });

  it('colors grades from easy to hard', () => {
    expect(gradeClass('trail')).toContain('emerald');
    expect(gradeClass('2b')).toContain('amber');
    expect(gradeClass('5a')).toContain('red');
  });

  it('counts days from a Bishkek calendar date', () => {
    // 2026-10-15 01:00 in Bishkek is still the 14th in UTC.
    const now = Date.parse('2026-10-14T19:00:00Z');
    expect(daysSince('2026-10-15', now)).toBe(0);
    expect(daysSince('2026-10-12', now)).toBe(3);
    expect(daysSince('2026-10-20', now)).toBe(0);
  });

  it('falls back to a question mark for unknown values', () => {
    expect(optionIcon(KINDS, 'hike')).toBe('icon-[mdi--hiking]');
    expect(optionIcon(KINDS, 'flying')).toBe('icon-[mdi--help-circle-outline]');
  });
});
