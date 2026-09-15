import { z } from 'zod';

export const envSchema = z.object({
  BOT_TOKEN: z.string().min(1, 'BOT_TOKEN is required'),
  POSTGRES_HOST: z.string().default('localhost'),
  POSTGRES_PORT: z.coerce.number().default(5432),
  POSTGRES_USER: z.string().default('parcel_bot'),
  POSTGRES_PASSWORD: z.string().default('change_me'),
  POSTGRES_DB: z.string().default('parcel_bot'),
  // URL of the Ermekten Telegram Mini App (the apps/tourism site). When set,
  // the bot exposes it via the chat menu button and a keyboard button.
  MINI_APP_URL: z.string().url().optional(),
});

export type Env = z.infer<typeof envSchema>;

export function validateEnv(config: Record<string, unknown>): Env {
  const parsed = envSchema.safeParse(config);
  if (!parsed.success) {
    throw new Error(`Invalid environment configuration: ${parsed.error.message}`);
  }
  return parsed.data;
}
