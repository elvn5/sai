import { GuideDifficulty, GuideSeason, Locale } from '@app/database';
import { z } from 'zod';

export const createGuideSchema = z.object({
  locale: z.nativeEnum(Locale),
  slug: z
    .string()
    .min(1)
    .max(255)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Slug must be lowercase, alphanumeric, hyphen-separated'),
  title: z.string().min(1).max(255),
  excerpt: z.string().max(500).nullable().optional(),
  content: z.string().min(1).max(20000),
  difficulty: z.nativeEnum(GuideDifficulty).optional().default(GuideDifficulty.EASY),
  season: z.nativeEnum(GuideSeason).optional().default(GuideSeason.ALL_YEAR),
  equipmentChecklist: z.array(z.string().min(1).max(255)).max(50).optional().default([]),
  coverImageUrl: z.string().max(2000).nullable().optional(),
});

export const updateGuideSchema = createGuideSchema.partial();

export type CreateGuideDto = z.infer<typeof createGuideSchema>;
export type UpdateGuideDto = z.infer<typeof updateGuideSchema>;
