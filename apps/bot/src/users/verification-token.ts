import { createHash } from 'node:crypto';

// Mirrors apps/api's hashVerificationCode — the bot reads the same
// verificationCodeHash column the API writes, so the hashing must match.
export function hashVerificationToken(token: string): string {
  return createHash('sha256').update(token).digest('hex');
}
