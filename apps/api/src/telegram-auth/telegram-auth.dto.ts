import { z } from 'zod';

export const telegramWidgetLoginSchema = z.object({
  id: z.number(),
  first_name: z.string(),
  last_name: z.string().optional(),
  username: z.string().optional(),
  photo_url: z.string().optional(),
  auth_date: z.number(),
  hash: z.string(),
});

export const telegramMiniAppLoginSchema = z.object({
  initData: z.string().min(1),
});

export type TelegramWidgetLoginDto = z.infer<typeof telegramWidgetLoginSchema>;
export type TelegramMiniAppLoginDto = z.infer<typeof telegramMiniAppLoginSchema>;
