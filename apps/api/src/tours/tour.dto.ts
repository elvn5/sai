import { Locale } from '@app/database';
import { z } from 'zod';

export const createTourSchema = z.object({
  locale: z.nativeEnum(Locale),
  slug: z
    .string()
    .min(1)
    .max(255)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Slug must be lowercase, alphanumeric, hyphen-separated'),
  title: z.string().min(1).max(255),
  excerpt: z.string().max(500).nullable().optional(),
  content: z.string().min(1).max(20000),
  price: z.number().int().nonnegative().nullable().optional(),
  durationDays: z.number().int().positive().nullable().optional(),
  location: z.string().max(255).nullable().optional(),
  coverImageUrl: z.string().max(2000).nullable().optional(),
});

export const updateTourSchema = createTourSchema.partial();

export type CreateTourDto = z.infer<typeof createTourSchema>;
export type UpdateTourDto = z.infer<typeof updateTourSchema>;
