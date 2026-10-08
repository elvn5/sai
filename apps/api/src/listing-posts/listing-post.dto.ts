import { paginationQuerySchema, photosSchema } from '@/common/listing.schemas';
import { ReviewTargetType } from '@app/database';
import { z } from 'zod';

export const POST_MAX_PHOTOS = 10;

export const listingPostsQuerySchema = paginationQuerySchema.extend({
  targetType: z.nativeEnum(ReviewTargetType),
  targetId: z.string().uuid(),
});

export const createListingPostSchema = z
  .object({
    targetType: z.nativeEnum(ReviewTargetType),
    targetId: z.string().uuid(),
    text: z.string().trim().max(3000).default(''),
    photos: photosSchema(POST_MAX_PHOTOS).default([]),
  })
  .refine((data) => data.text.length > 0 || data.photos.length > 0, {
    message: 'A post needs text or at least one photo',
    path: ['text'],
  });

export type ListingPostsQueryDto = z.infer<typeof listingPostsQuerySchema>;
export type CreateListingPostDto = z.infer<typeof createListingPostSchema>;
