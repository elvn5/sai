import { createHash, randomInt, timingSafeEqual } from 'node:crypto';

export const VERIFICATION_CODE_TTL_MS = 10 * 60 * 1000;
export const VERIFICATION_RESEND_COOLDOWN_MS = 60 * 1000;
export const VERIFICATION_MAX_ATTEMPTS = 5;

export function generateVerificationCode(): string {
  return randomInt(0, 1_000_000).toString().padStart(6, '0');
}

export function hashVerificationCode(code: string): string {
  return createHash('sha256').update(code).digest('hex');
}

export function verificationCodeMatches(code: string, hash: string): boolean {
  const provided = Buffer.from(hashVerificationCode(code));
  const expected = Buffer.from(hash);
  return provided.length === expected.length && timingSafeEqual(provided, expected);
}
