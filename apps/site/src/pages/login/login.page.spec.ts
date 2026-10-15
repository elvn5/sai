import { normalizeAwardCode } from '@/pages/achievements/award.page';
import { safeRedirect } from '@/pages/login/login.page';

describe('sign-in helpers', () => {
  it('returns only to paths on this site', () => {
    expect(safeRedirect('/en/tours/ala-kul?x=1')).toBe('/en/tours/ala-kul?x=1');
    expect(safeRedirect('//evil.example/x')).toBeNull();
    expect(safeRedirect('https://evil.example')).toBeNull();
    expect(safeRedirect(undefined)).toBeNull();
  });

  it('reads award codes however they are typed', () => {
    expect(normalizeAwardCode(' abcd-efgh ')).toBe('ABCDEFGH');
  });
});
