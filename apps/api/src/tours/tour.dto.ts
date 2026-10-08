import { contactFieldsSchema } from '@/common/listing.schemas';
import { CancellationPolicy, Destination, Locale } from '@app/database';
import { z } from 'zod';

const tourFieldsSchema = z
  .object({
    locale: z.nativeEnum(Locale),
    slug: z
      .string()
      .min(1)
      .max(255)
      .regex(
        /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
        'Slug must be lowercase, alphanumeric, hyphen-separated',
      ),
    title: z.string().min(1).max(255),
    excerpt: z.string().max(500).nullable().optional(),
    content: z.string().min(1).max(20000),
    price: z.number().int().nonnegative().nullable().optional(),
    durationDays: z.number().int().positive().nullable().optional(),
    location: z.string().max(255).nullable().optional(),
    startDate: z.string().date(),
    endDate: z.string().date(),
    destination: z.nativeEnum(Destination).nullable().optional(),
    cancellationPolicy: z.nativeEnum(CancellationPolicy).nullable().optional(),
    coverImageUrl: z.string().max(2000).nullable().optional(),
  })
  .merge(contactFieldsSchema);

export const createTourSchema = tourFieldsSchema.refine((data) => data.endDate >= data.startDate, {
  message: 'endDate must be on or after startDate',
  path: ['endDate'],
});

export const updateTourSchema = tourFieldsSchema.partial();

export type CreateTourDto = z.infer<typeof createTourSchema>;
export type UpdateTourDto = z.infer<typeof updateTourSchema>;

export const tourListQuerySchema = z.object({
  locale: z.nativeEnum(Locale),
  destination: z.nativeEnum(Destination).optional(),
});

export type TourListQueryDto = z.infer<typeof tourListQuerySchema>;
