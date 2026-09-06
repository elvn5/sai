import { z } from 'zod';

export const updateSupportInfoSchema = z.object({
  message: z.string().min(1).max(4000),
  phone: z.string().max(50).nullable().optional(),
  telegramContact: z.string().max(255).nullable().optional(),
  email: z.string().email().max(255).nullable().optional(),
  workingHours: z.string().max(255).nullable().optional(),
});

export type UpdateSupportInfoDto = z.infer<typeof updateSupportInfoSchema>;
