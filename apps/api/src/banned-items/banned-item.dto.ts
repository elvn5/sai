import { z } from 'zod';

export const createBannedItemSchema = z.object({
  name: z.string().min(1).max(255),
  description: z.string().max(2000).nullable().optional(),
  isActive: z.boolean().optional().default(true),
  sortOrder: z.number().int().optional().default(0),
});

export const updateBannedItemSchema = createBannedItemSchema.partial();

export type CreateBannedItemDto = z.infer<typeof createBannedItemSchema>;
export type UpdateBannedItemDto = z.infer<typeof updateBannedItemSchema>;
