import { createHash, randomBytes, randomInt, timingSafeEqual } from 'node:crypto';

export const VERIFICATION_CODE_TTL_MS = 10 * 60 * 1000;
export const VERIFICATION_RESEND_COOLDOWN_MS = 60 * 1000;
export const VERIFICATION_MAX_ATTEMPTS = 5;

export function generateVerificationCode(): string {
  return randomInt(0, 1_000_000).toString().padStart(6, '0');
}

// Longer, URL-safe token used for the Telegram deep-link flow — unlike the
// 6-digit email code, this is never typed by hand, so it doesn't need to be
// short.
export function generateVerificationToken(): string {
  return randomBytes(24).toString('hex');
}

export function hashVerificationCode(code: string): string {
  return createHash('sha256').update(code).digest('hex');
}

export function verificationCodeMatches(code: string, hash: string): boolean {
  const provided = Buffer.from(hashVerificationCode(code));
  const expected = Buffer.from(hash);
  return provided.length === expected.length && timingSafeEqual(provided, expected);
}
