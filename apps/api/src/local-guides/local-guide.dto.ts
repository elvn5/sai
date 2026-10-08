import { GUIDE_LANGUAGES, GUIDE_SPECIALIZATIONS } from '@/common/listing-options';
import {
  contactFieldsSchema,
  mediaUrlSchema,
  paginationQuerySchema,
  photosSchema,
  queryBooleanSchema,
  slugSchema,
} from '@/common/listing.schemas';
import { Destination } from '@app/database';
import { z } from 'zod';

export const LOCAL_GUIDE_MAX_PHOTOS = 20;

export const createLocalGuideSchema = z
  .object({
    slug: slugSchema,
    name: z.string().trim().min(1).max(255),
    headline: z.string().trim().max(160).nullable().optional(),
    bio: z.string().min(1).max(20000),
    avatarUrl: mediaUrlSchema.nullable().optional(),
    coverImageUrl: mediaUrlSchema.nullable().optional(),
    photos: photosSchema(LOCAL_GUIDE_MAX_PHOTOS).default([]),
    languages: z.array(z.enum(GUIDE_LANGUAGES)).min(1).max(GUIDE_LANGUAGES.length),
    specializations: z
      .array(z.enum(GUIDE_SPECIALIZATIONS))
      .min(1)
      .max(GUIDE_SPECIALIZATIONS.length),
    destinations: z.array(z.nativeEnum(Destination)).min(1),
    pricePerDay: z.number().int().nonnegative().nullable().optional(),
    experienceYears: z.number().int().min(0).max(80).nullable().optional(),
    hasVehicle: z.boolean().default(false),
  })
  .merge(contactFieldsSchema);

export const updateLocalGuideSchema = createLocalGuideSchema.partial();

export const LOCAL_GUIDE_SORTS = [
  'rating',
  'price_asc',
  'price_desc',
  'experience',
  'newest',
] as const;

export const localGuideListQuerySchema = paginationQuerySchema.extend({
  destination: z.nativeEnum(Destination).optional(),
  language: z.enum(GUIDE_LANGUAGES).optional(),
  specialization: z.enum(GUIDE_SPECIALIZATIONS).optional(),
  maxPrice: z.coerce.number().int().nonnegative().optional(),
  minRating: z.coerce.number().min(0).max(5).optional(),
  hasVehicle: queryBooleanSchema,
  search: z.string().trim().max(100).optional(),
  sort: z.enum(LOCAL_GUIDE_SORTS).default('rating'),
});

export type CreateLocalGuideDto = z.infer<typeof createLocalGuideSchema>;
export type UpdateLocalGuideDto = z.infer<typeof updateLocalGuideSchema>;
export type LocalGuideListQueryDto = z.infer<typeof localGuideListQuerySchema>;
