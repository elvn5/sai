import { Locale } from '@app/database';
import { z } from 'zod';

export const createBlogPostSchema = z.object({
  locale: z.nativeEnum(Locale),
  slug: z
    .string()
    .min(1)
    .max(255)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Slug must be lowercase, alphanumeric, hyphen-separated'),
  title: z.string().min(1).max(255),
  excerpt: z.string().max(500).nullable().optional(),
  content: z.string().min(1),
  coverImageUrl: z.string().max(2000).nullable().optional(),
  seoTitle: z.string().max(255).nullable().optional(),
  seoDescription: z.string().max(500).nullable().optional(),
  isPublished: z.boolean().optional().default(false),
});

export const updateBlogPostSchema = createBlogPostSchema.partial();

export type CreateBlogPostDto = z.infer<typeof createBlogPostSchema>;
export type UpdateBlogPostDto = z.infer<typeof updateBlogPostSchema>;
