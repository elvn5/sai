import { z } from 'zod';

export const upsertPageSeoSchema = z.object({
  title: z.string().max(255).nullable().optional(),
  description: z.string().max(500).nullable().optional(),
  ogImageUrl: z.string().max(2000).nullable().optional(),
});

export type UpsertPageSeoDto = z.infer<typeof upsertPageSeoSchema>;
