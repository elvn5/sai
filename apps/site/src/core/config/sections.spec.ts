import { isMigrated } from '@/core/config/sections';

describe('isMigrated', () => {
  const sections = ['/', '/blog'];

  it('matches a section and its pages', () => {
    expect(isMigrated('/blog', sections)).toBe(true);
    expect(isMigrated('/blog/hello', sections)).toBe(true);
  });

  it('treats "/" as the home page only', () => {
    expect(isMigrated('/', sections)).toBe(true);
    expect(isMigrated('/tours', sections)).toBe(false);
    expect(isMigrated('/blogger', sections)).toBe(false);
  });
});
