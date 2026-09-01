import { ParcelStatus } from '@app/database';
import { z } from 'zod';

export const createParcelSchema = z.object({
  userId: z.string().uuid(),
  trackNumber: z.string().min(1).max(255),
  description: z.string().max(2000).nullable().optional(),
  status: z.nativeEnum(ParcelStatus).optional().default(ParcelStatus.CREATED),
  weightKg: z.string().max(20).nullable().optional(),
  comment: z.string().max(2000).nullable().optional(),
});

export const updateParcelSchema = createParcelSchema.omit({ userId: true }).partial();

export type CreateParcelDto = z.infer<typeof createParcelSchema>;
export type UpdateParcelDto = z.infer<typeof updateParcelSchema>;
