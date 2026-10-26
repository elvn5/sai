import { movedSectionRedirect, pausedSectionRedirect } from '@/server/paused-sections';

describe('pausedSectionRedirect', () => {
  it('sends a paused section to the closest one, keeping the language', () => {
    expect(pausedSectionRedirect('/hotels')).toBe('/tours');
    expect(pausedSectionRedirect('/en/hotels/some-hotel?x=1')).toBe('/en/tours');
    expect(pausedSectionRedirect('/ky/trails/ala-archa')).toBe('/ky/tours');
    expect(pausedSectionRedirect('/de/local-guides')).toBe('/de/tours');
    expect(pausedSectionRedirect('/payments/return?order=1')).toBe('/tours');
    expect(pausedSectionRedirect('/en/hikes')).toBe('/en/tours');
  });

  it('leaves the other addresses alone', () => {
    expect(pausedSectionRedirect('/')).toBeNull();
    expect(pausedSectionRedirect('/tours/hotels-of-issyk-kul')).toBeNull();
    expect(pausedSectionRedirect('/en/outings')).toBeNull();
    expect(pausedSectionRedirect('/guidesx')).toBeNull();
  });
});

describe('movedSectionRedirect', () => {
  it('sends outings to tours for good, keeping the language', () => {
    expect(movedSectionRedirect('/outings')).toBe('/tours');
    expect(movedSectionRedirect('/outings?free=1')).toBe('/tours');
    expect(movedSectionRedirect('/en/outings/0b6c1a6e-5f3e-4f1e-9a4b-1c2d3e4f5a6b')).toBe(
      '/en/tours',
    );
    expect(movedSectionRedirect('/ky/outings/new')).toBe('/ky/tours');
  });

  it('leaves the other addresses alone', () => {
    expect(movedSectionRedirect('/tours')).toBeNull();
    expect(movedSectionRedirect('/outingsx')).toBeNull();
    expect(movedSectionRedirect('/hotels')).toBeNull();
  });
});
