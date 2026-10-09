import { z } from 'zod';

export const envSchema = z.object({
  BOT_TOKEN: z.string().min(1, 'BOT_TOKEN is required'),
  // users service's internal gRPC API (accounts, support contacts).
  USERS_GRPC_ADDR: z.string().min(1, 'USERS_GRPC_ADDR is required'),
  // Overrides the location of services/proto/users/v1/users.proto.
  USERS_PROTO_PATH: z.string().optional(),
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
