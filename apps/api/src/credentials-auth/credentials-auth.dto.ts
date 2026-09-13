import { z } from 'zod';

export const registerSchema = z
  .object({
    email: z.string().email().max(255).optional(),
    phone: z.string().min(5).max(30).optional(),
    password: z.string().min(8).max(100),
    firstName: z.string().min(1).max(100).optional(),
  })
  .refine((data) => Boolean(data.email || data.phone), {
    message: 'Email or phone is required',
    path: ['email'],
  });

export const loginSchema = z.object({
  identifier: z.string().min(1).max(255),
  password: z.string().min(1).max(100),
});

export type RegisterDto = z.infer<typeof registerSchema>;
export type LoginDto = z.infer<typeof loginSchema>;
