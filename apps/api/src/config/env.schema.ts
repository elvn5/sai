import { z } from 'zod';

export const envSchema = z.object({
  API_PORT: z.coerce.number().default(3001),
  POSTGRES_HOST: z.string().default('localhost'),
  POSTGRES_PORT: z.coerce.number().default(5432),
  POSTGRES_USER: z.string().default('parcel_bot'),
  POSTGRES_PASSWORD: z.string().default('change_me'),
  POSTGRES_DB: z.string().default('parcel_bot'),
  JWT_SECRET: z.string().min(16, 'JWT_SECRET must be at least 16 characters'),
  JWT_EXPIRES_IN: z.string().default('12h'),
  ADMIN_PASSWORD: z.string().min(1, 'ADMIN_PASSWORD is required'),
  BOT_TOKEN: z.string().min(1, 'BOT_TOKEN is required'),
  ADMIN_ORIGIN: z.string().default('http://localhost:3000'),
  CLIENT_ORIGIN: z.string().default('http://localhost:3002'),
  UPLOADS_DIR: z.string().optional(),
  // Email verification codes (Resend). Optional so the app can still boot
  // without them — registration by email just fails with a clear error
  // until they're set, instead of crash-looping the whole API.
  RESEND_API_KEY: z.string().optional(),
  EMAIL_FROM: z.string().default('Ermekten <onboarding@resend.dev>'),
  // Phone verification codes (Twilio WhatsApp). Same optionality as above.
  TWILIO_ACCOUNT_SID: z.string().optional(),
  TWILIO_AUTH_TOKEN: z.string().optional(),
  TWILIO_WHATSAPP_FROM: z.string().optional(),
});

export type Env = z.infer<typeof envSchema>;

export function validateEnv(config: Record<string, unknown>): Env {
  const parsed = envSchema.safeParse(config);
  if (!parsed.success) {
    throw new Error(`Invalid environment configuration: ${parsed.error.message}`);
  }
  return parsed.data;
}
