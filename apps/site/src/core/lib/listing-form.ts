import { z } from 'zod/mini';

// Contact rules of listing forms (the API's own patterns).
export const PHONE_PATTERN = /^\+?[0-9\s\-()]{5,32}$/;
export const TELEGRAM_PATTERN = /^@?[A-Za-z0-9_]{5,32}$/;

// Empty, or matching `pattern`.
export function optionalPattern(pattern: RegExp, message: string): z.ZodMiniType<string> {
  return z
    .string()
    .check(z.refine((value) => value.trim() === '' || pattern.test(value.trim()), message));
}

export function optionalUrl(message: string): z.ZodMiniType<string> {
  return z.string().check(
    z.refine((value) => {
      if (value.trim() === '') return true;
      try {
        return ['http:', 'https:'].includes(new URL(value.trim()).protocol);
      } catch {
        return false;
      }
    }, message),
  );
}

// HTML from the editor with some visible text.
export function requiredHtml(message: string): z.ZodMiniType<string> {
  return z.string().check(z.refine((html) => html.replace(/<[^>]*>/g, '').trim() !== '', message));
}

export function emptyToNull(value: string | null | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

// "@username" → "username" (stored without the @).
export function telegramUsername(value: string): string | null {
  return emptyToNull(value)?.replace(/^@/, '') ?? null;
}
