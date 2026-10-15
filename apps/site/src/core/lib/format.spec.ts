import { fileSize, localizedText, personName } from '@/core/lib/format';
import { slugify } from '@/core/lib/slugify';

describe('format', () => {
  it('picks the language of admin texts, falling back to Russian', () => {
    expect(localizedText({ ru: 'Перевал', en: 'Pass' }, 'en')).toBe('Pass');
    expect(localizedText({ ru: 'Перевал' }, 'de')).toBe('Перевал');
  });

  it('names people by first name, else @username', () => {
    expect(personName({ id: '1', firstName: 'Aida', username: 'aida', avatarUrl: null })).toBe(
      'Aida',
    );
    expect(personName({ id: '1', firstName: null, username: 'aida', avatarUrl: null })).toBe(
      '@aida',
    );
    expect(personName({ id: '1', firstName: null, username: null, avatarUrl: null })).toBe('—');
  });

  it('formats file sizes', () => {
    expect(fileSize(900)).toBe('900 B');
    expect(fileSize(2048)).toBe('2 KB');
    expect(fileSize(3.5 * 1024 * 1024)).toBe('3.5 MB');
  });
});

describe('slugify', () => {
  it('transliterates Russian and Kyrgyz and adds a suffix', () => {
    expect(slugify('Озеро Сон-Көл!', () => 0.123456789)).toBe('ozero-son-kol-4fzzzx');
    expect(slugify('Жайлоо ңү', () => 0.5)).toMatch(/^zhayloo-ngu-[a-z0-9]+$/);
  });

  it('still makes a slug from symbols only', () => {
    expect(slugify('!!!', () => 0.5)).toMatch(/^[a-z0-9]+$/);
  });
});
