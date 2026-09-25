import { z } from 'zod';

export const createGalleryItemSchema = z.object({
  imageUrl: z.string().min(1).max(2000),
  location: z.string().max(255).nullable().optional(),
  caption: z.string().max(500).nullable().optional(),
  altText: z.string().max(255).nullable().optional(),
});

export const updateGalleryItemSchema = createGalleryItemSchema.partial();

export type CreateGalleryItemDto = z.infer<typeof createGalleryItemSchema>;
export type UpdateGalleryItemDto = z.infer<typeof updateGalleryItemSchema>;
