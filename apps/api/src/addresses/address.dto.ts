import { z } from 'zod';

export const createAddressSchema = z.object({
  title: z.string().min(1).max(255),
  fullAddress: z.string().min(1).max(2000),
  comment: z.string().max(2000).nullable().optional(),
  isActive: z.boolean().optional().default(true),
  sortOrder: z.number().int().optional().default(0),
});

export const updateAddressSchema = createAddressSchema.partial();

export type CreateAddressDto = z.infer<typeof createAddressSchema>;
export type UpdateAddressDto = z.infer<typeof updateAddressSchema>;
