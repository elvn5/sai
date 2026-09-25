import { z } from 'zod';

const ALLOWED_UPLOAD_MIME_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const;

export const presignedUploadSchema = z.object({
  filename: z.string().min(1).max(255),
  contentType: z.enum(ALLOWED_UPLOAD_MIME_TYPES),
  folder: z
    .string()
    .max(100)
    .regex(/^[a-z0-9-]+$/, 'folder must be lowercase alphanumeric with hyphens')
    .optional(),
});

export const deleteFileSchema = z.object({
  key: z.string().min(1).max(500),
});

export type PresignedUploadDto = z.infer<typeof presignedUploadSchema>;
export type DeleteFileDto = z.infer<typeof deleteFileSchema>;
