import type { ReviewTargetType, ReviewsPage } from '@/core/api/types';
import { type HttpResourceRef, httpResource } from '@angular/common/http';
import type { Signal } from '@angular/core';

export const REVIEWS_PAGE_SIZE = 10;

// The first page of a listing's reviews (rendered on the server, shown by
// ReviewsSection and summarized in the page's JSON-LD).
export function reviewsResource(
  targetType: ReviewTargetType,
  targetId: Signal<string | undefined>,
): HttpResourceRef<ReviewsPage | undefined> {
  return httpResource<ReviewsPage>(() => {
    const id = targetId();
    return id
      ? {
          url: '/reviews',
          params: { targetType, targetId: id, limit: REVIEWS_PAGE_SIZE, offset: 0 },
        }
      : undefined;
  });
}
