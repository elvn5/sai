import { checkDevPassword, normalizeHash, sha256Hex } from '@/core/feature-flags/dev-menu-password';

// SHA-256 of "open sesame".
const HASH = '41ef4bb0b23661e66301aac36066912dac037827b4ae63a7b1165a5aa93ed4eb';

describe('sha256Hex', () => {
  it('hashes text as lowercase hex', async () => {
    expect(await sha256Hex('')).toBe(
      'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    );
    expect(await sha256Hex('open sesame')).toBe(HASH);
  });
});

describe('normalizeHash', () => {
  it('accepts 64 hex characters in any case', () => {
    expect(normalizeHash(` ${HASH.toUpperCase()}\n`)).toBe(HASH);
  });

  it('rejects anything else', () => {
    expect(normalizeHash(undefined)).toBe('');
    expect(normalizeHash('')).toBe('');
    expect(normalizeHash('open sesame')).toBe('');
    expect(normalizeHash(HASH.slice(1))).toBe('');
  });
});

describe('checkDevPassword', () => {
  it('lets the right password in', async () => {
    expect(await checkDevPassword('open sesame', HASH)).toBe(true);
    expect(await checkDevPassword('open sesame', HASH.toUpperCase())).toBe(true);
  });

  it('keeps a wrong or empty password out', async () => {
    expect(await checkDevPassword('open sesame!', HASH)).toBe(false);
    expect(await checkDevPassword('', HASH)).toBe(false);
  });

  it('keeps the menu closed without a hash in the config', async () => {
    expect(await checkDevPassword('open sesame', '')).toBe(false);
    expect(await checkDevPassword('anything', 'not-a-hash')).toBe(false);
  });
});
