import { z } from 'zod';

export const postLanguages = ['ru', 'ky', 'en'] as const;
export const postStatuses = ['draft', 'published'] as const;

export const createPostSchema = z.object({
  title: z.string().min(1).max(255),
  slug: z
    .string()
    .min(1)
    .max(255)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'slug: только строчные латинские буквы, цифры и дефис')
    .optional(),
  excerpt: z.string().max(500).nullable().optional(),
  content: z.string().min(1),
  coverImageUrl: z.string().url().max(2000).nullable().optional(),
  language: z.enum(postLanguages).optional().default('ru'),
  status: z.enum(postStatuses).optional().default('draft'),
});

export const updatePostSchema = createPostSchema.partial();

export type CreatePostDto = z.infer<typeof createPostSchema>;
export type UpdatePostDto = z.infer<typeof updatePostSchema>;
