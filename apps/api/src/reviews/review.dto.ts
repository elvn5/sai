import { paginationQuerySchema, photosSchema } from '@/common/listing.schemas';
import { ReviewTargetType } from '@app/database';
import { z } from 'zod';

export const REVIEW_MAX_PHOTOS = 6;

export const reviewTargetQuerySchema = paginationQuerySchema.extend({
  targetType: z.nativeEnum(ReviewTargetType),
  targetId: z.string().uuid(),
});

export const createReviewSchema = z.object({
  targetType: z.nativeEnum(ReviewTargetType),
  targetId: z.string().uuid(),
  rating: z.number().int().min(1).max(5),
  text: z.string().trim().min(10).max(3000),
  photos: photosSchema(REVIEW_MAX_PHOTOS).default([]),
});

export type ReviewTargetQueryDto = z.infer<typeof reviewTargetQuerySchema>;
export type CreateReviewDto = z.infer<typeof createReviewSchema>;
