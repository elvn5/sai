import { z } from 'zod';

// Media is either a full R2 public URL or a legacy local upload path.
export const mediaUrlSchema = z
  .string()
  .max(2000)
  .regex(/^(https?:\/\/|\/uploads\/)/, 'Must be an uploaded file URL');

export function photosSchema(max: number) {
  return z.array(mediaUrlSchema).max(max);
}

export const slugSchema = z
  .string()
  .min(1)
  .max(255)
  .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Slug must be lowercase, alphanumeric, hyphen-separated');

const phoneSchema = z
  .string()
  .max(32)
  .regex(/^\+?[0-9\s\-()]{5,32}$/, 'Invalid phone number');

// Telegram usernames are 5–32 chars; the leading @ is optional on input.
const telegramSchema = z
  .string()
  .max(33)
  .regex(/^@?[A-Za-z0-9_]{5,32}$/, 'Invalid Telegram username')
  .transform((value) => value.replace(/^@/, ''));

export const contactFieldsSchema = z.object({
  phone: phoneSchema.nullable().optional(),
  whatsapp: phoneSchema.nullable().optional(),
  telegram: telegramSchema.nullable().optional(),
});

export const paginationQuerySchema = z.object({
  limit: z.coerce.number().int().min(1).max(50).default(20),
  offset: z.coerce.number().int().min(0).default(0),
});

// Query strings carry booleans as text.
export const queryBooleanSchema = z
  .enum(['true', 'false'])
  .transform((value) => value === 'true')
  .optional();

// Comma-separated list in a query string, e.g. ?amenities=wifi,breakfast
export function queryListSchema<T extends [string, ...string[]]>(values: T) {
  return z
    .string()
    .optional()
    .transform((raw) => (raw ? raw.split(',').filter(Boolean) : []))
    .pipe(z.array(z.enum(values)));
}

export interface Paginated<T> {
  items: T[];
  total: number;
}
