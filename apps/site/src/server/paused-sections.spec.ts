import { pausedSectionRedirect } from '@/server/paused-sections';

describe('pausedSectionRedirect', () => {
  it('sends a paused section to the closest one, keeping the language', () => {
    expect(pausedSectionRedirect('/hotels')).toBe('/tours');
    expect(pausedSectionRedirect('/en/hotels/some-hotel?x=1')).toBe('/en/tours');
    expect(pausedSectionRedirect('/ky/trails/ala-archa')).toBe('/ky/outings');
    expect(pausedSectionRedirect('/de/local-guides')).toBe('/de/tours');
    expect(pausedSectionRedirect('/payments/return?order=1')).toBe('/tours');
    expect(pausedSectionRedirect('/en/hikes')).toBe('/en/outings');
  });

  it('leaves the other addresses alone', () => {
    expect(pausedSectionRedirect('/')).toBeNull();
    expect(pausedSectionRedirect('/tours/hotels-of-issyk-kul')).toBeNull();
    expect(pausedSectionRedirect('/en/outings')).toBeNull();
    expect(pausedSectionRedirect('/guidesx')).toBeNull();
  });
});
