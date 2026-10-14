import { isLang, localizedPath } from '@/shared/i18n/langs';

describe('localizedPath', () => {
  it('keeps Russian unprefixed and prefixes the other languages', () => {
    expect(localizedPath('ru', '/tours')).toBe('/tours');
    expect(localizedPath('en', '/tours')).toBe('/en/tours');
    expect(localizedPath('ky', '/')).toBe('/ky');
  });

  it('knows the site languages', () => {
    expect(isLang('de')).toBe(true);
    expect(isLang('fr')).toBe(false);
  });
});
