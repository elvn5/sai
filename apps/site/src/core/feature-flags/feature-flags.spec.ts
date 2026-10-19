import {
  FEATURE_FLAGS,
  FEATURE_FLAGS_KEY,
  FeatureFlags,
  parseOverrides,
} from '@/core/feature-flags/feature-flags';
import { PLATFORM_ID } from '@angular/core';
import { TestBed } from '@angular/core/testing';

function flags(platform: 'browser' | 'server' = 'browser'): FeatureFlags {
  TestBed.configureTestingModule({ providers: [{ provide: PLATFORM_ID, useValue: platform }] });
  const service = TestBed.inject(FeatureFlags);
  // Runs afterNextRender, where the saved switches load.
  TestBed.tick();
  return service;
}

describe('parseOverrides', () => {
  it('keeps known flags with a boolean value', () => {
    expect(
      parseOverrides('{"favorites":false,"booking":true,"unknown":false,"tourDates":"no"}'),
    ).toEqual({
      favorites: false,
      booking: true,
    });
  });

  it('ignores missing or broken data', () => {
    expect(parseOverrides(null)).toEqual({});
    expect(parseOverrides('not json')).toEqual({});
    expect(parseOverrides('[1,2]')).toEqual({});
    expect(parseOverrides('null')).toEqual({});
  });
});

describe('FeatureFlags', () => {
  beforeEach(() => localStorage.clear());

  it('has everything on by default', () => {
    const service = flags();
    expect(FEATURE_FLAGS.every((flag) => service.isOn(flag))).toBe(true);
    expect(service.changedCount()).toBe(0);
  });

  it('switches a flag off and on, keeping only the changes', () => {
    const service = flags();
    service.set('favorites', false);
    expect(service.isOn('favorites')).toBe(false);
    expect(service.isOn('booking')).toBe(true);
    expect(JSON.parse(localStorage.getItem(FEATURE_FLAGS_KEY) ?? '{}')).toEqual({
      favorites: false,
    });

    service.set('favorites', true);
    expect(service.isOn('favorites')).toBe(true);
    expect(service.changedCount()).toBe(0);
    expect(localStorage.getItem(FEATURE_FLAGS_KEY)).toBe('{}');
  });

  it('loads the saved switches after the first render', () => {
    localStorage.setItem(FEATURE_FLAGS_KEY, JSON.stringify({ tourGallery: false }));
    const service = flags();
    expect(service.isOn('tourGallery')).toBe(false);
    expect(service.changedCount()).toBe(1);
  });

  it('resets everything', () => {
    const service = flags();
    service.set('booking', false);
    service.set('viewTransitions', false);
    service.reset();
    expect(service.changedCount()).toBe(0);
    expect(service.isOn('booking')).toBe(true);
    expect(localStorage.getItem(FEATURE_FLAGS_KEY)).toBe('{}');
  });

  it('stays all on and stores nothing on the server', () => {
    localStorage.setItem(FEATURE_FLAGS_KEY, JSON.stringify({ favorites: false }));
    const service = flags('server');
    expect(service.isOn('favorites')).toBe(true);
    service.set('booking', false);
    expect(localStorage.getItem(FEATURE_FLAGS_KEY)).toBe('{"favorites":false}');
  });

  it('works when storage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    const service = flags();
    expect(service.isOn('favorites')).toBe(true);
    service.set('favorites', false);
    expect(service.isOn('favorites')).toBe(false);
    vi.restoreAllMocks();
  });
});
