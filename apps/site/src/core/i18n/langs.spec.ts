import { isLang, localizedPath, splitLang } from '@/core/i18n/langs';

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

describe('splitLang', () => {
  it('takes the language prefix off the path', () => {
    expect(splitLang('/en/tours/altyn-arashan?page=2#top')).toEqual({
      lang: 'en',
      path: '/tours/altyn-arashan',
    });
    expect(splitLang('/de')).toEqual({ lang: 'de', path: '/' });
  });

  it('reads unprefixed paths as Russian', () => {
    expect(splitLang('/tours?page=2')).toEqual({ lang: 'ru', path: '/tours' });
    expect(splitLang('/')).toEqual({ lang: 'ru', path: '/' });
    // A section that merely starts with a language code.
    expect(splitLang('/entries')).toEqual({ lang: 'ru', path: '/entries' });
  });
});
