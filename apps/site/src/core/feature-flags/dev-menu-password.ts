// The developer menu's password is never in the bundle: the SSR server
// hands the browser its SHA-256 (env DEV_MENU_PASSWORD_HASH, hex) through
// APP_CONFIG, and what the visitor types is hashed here and compared.

export async function sha256Hex(text: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text));
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('');
}

// A hash in the config's format (64 hex characters), lowercased; '' when
// unset or malformed — the menu then stays closed.
export function normalizeHash(hash: string | undefined): string {
  const value = (hash ?? '').trim().toLowerCase();
  return /^[0-9a-f]{64}$/.test(value) ? value : '';
}

export async function checkDevPassword(password: string, hash: string): Promise<boolean> {
  const expected = normalizeHash(hash);
  // crypto.subtle exists only on https and localhost.
  if (!expected || !password || !globalThis.crypto?.subtle) return false;
  return (await sha256Hex(password)) === expected;
}
