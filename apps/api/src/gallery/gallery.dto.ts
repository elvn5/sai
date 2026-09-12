import { z } from 'zod';

export const createGalleryItemSchema = z.object({
  imageUrl: z.string().min(1).max(2000),
  caption: z.string().max(500).nullable().optional(),
  altText: z.string().max(255).nullable().optional(),
  sortOrder: z.number().int().optional().default(0),
  isPublished: z.boolean().optional().default(true),
});

export const updateGalleryItemSchema = createGalleryItemSchema.partial();

export type CreateGalleryItemDto = z.infer<typeof createGalleryItemSchema>;
export type UpdateGalleryItemDto = z.infer<typeof updateGalleryItemSchema>;
