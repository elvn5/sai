import { readResetToken } from '@/pages/login/reset-password.page';

describe('readResetToken', () => {
  const token = 'a'.repeat(48);

  it('reads the token from the fragment', () => {
    expect(readResetToken(`#${token}`)).toBe(token);
    expect(readResetToken(token)).toBe(token);
  });

  it('refuses anything that cannot be a token', () => {
    expect(readResetToken('')).toBeNull();
    expect(readResetToken('#')).toBeNull();
    expect(readResetToken('#short')).toBeNull();
    expect(readResetToken(`#${token}&x=1`)).toBeNull();
    expect(readResetToken(`#${'a'.repeat(129)}`)).toBeNull();
  });
});
