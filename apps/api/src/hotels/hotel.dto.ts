import { HOTEL_AMENITIES } from '@/common/listing-options';
import {
  contactFieldsSchema,
  mediaUrlSchema,
  paginationQuerySchema,
  photosSchema,
  queryBooleanSchema,
  queryListSchema,
  slugSchema,
} from '@/common/listing.schemas';
import { CancellationPolicy, Destination, HotelType } from '@app/database';
import { z } from 'zod';

export const HOTEL_MAX_PHOTOS = 20;

const timeSchema = z.string().regex(/^([01]\d|2[0-3]):[0-5]\d$/, 'Expected HH:MM');

export const createHotelSchema = z
  .object({
    slug: slugSchema,
    name: z.string().trim().min(1).max(255),
    type: z.nativeEnum(HotelType),
    description: z.string().min(1).max(20000),
    destination: z.nativeEnum(Destination),
    address: z.string().max(255).nullable().optional(),
    latitude: z.number().min(-90).max(90).nullable().optional(),
    longitude: z.number().min(-180).max(180).nullable().optional(),
    priceFrom: z.number().int().nonnegative().nullable().optional(),
    stars: z.number().int().min(1).max(5).nullable().optional(),
    amenities: z.array(z.enum(HOTEL_AMENITIES)).max(HOTEL_AMENITIES.length).default([]),
    cancellationPolicy: z.nativeEnum(CancellationPolicy),
    checkInTime: timeSchema.nullable().optional(),
    checkOutTime: timeSchema.nullable().optional(),
    coverImageUrl: mediaUrlSchema.nullable().optional(),
    photos: photosSchema(HOTEL_MAX_PHOTOS).default([]),
    website: z.string().url().max(500).nullable().optional(),
  })
  .merge(contactFieldsSchema);

export const updateHotelSchema = createHotelSchema.partial();

export const HOTEL_SORTS = ['rating', 'price_asc', 'price_desc', 'newest'] as const;

export const hotelListQuerySchema = paginationQuerySchema.extend({
  destination: z.nativeEnum(Destination).optional(),
  type: z.nativeEnum(HotelType).optional(),
  maxPrice: z.coerce.number().int().nonnegative().optional(),
  minRating: z.coerce.number().min(0).max(5).optional(),
  amenities: queryListSchema([...HOTEL_AMENITIES]),
  freeCancellation: queryBooleanSchema,
  search: z.string().trim().max(100).optional(),
  sort: z.enum(HOTEL_SORTS).default('rating'),
});

export type CreateHotelDto = z.infer<typeof createHotelSchema>;
export type UpdateHotelDto = z.infer<typeof updateHotelSchema>;
export type HotelListQueryDto = z.infer<typeof hotelListQuerySchema>;
