import { passwordFitsBcrypt } from '@/core/auth/password';

describe('password length', () => {
  it('checks UTF-8 bytes for Latin, Cyrillic and emoji passwords', () => {
    for (const [char, limit] of [
      ['a', 72],
      ['я', 36],
      ['🏔', 18],
    ] as const) {
      expect(passwordFitsBcrypt(char.repeat(limit))).toBe(true);
      expect(passwordFitsBcrypt(char.repeat(limit + 1))).toBe(false);
    }
  });
});
