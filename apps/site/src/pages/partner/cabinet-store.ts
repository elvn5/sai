import type { Paginated, PartnerReview } from '@/core/api/types';
import { httpResource } from '@angular/common/http';
import { Injectable, computed } from '@angular/core';

// State the cabinet's frame and its pages share: the number of reviews
// still waiting for an answer (the menu badge), refreshed after a reply.
// Provided by PartnerLayout, so it lives as long as the cabinet is open.
@Injectable()
export class CabinetStore {
  private readonly unanswered = httpResource<Paginated<PartnerReview>>(() => ({
    url: '/reviews/mine',
    params: { limit: 1, unanswered: true },
  }));

  readonly unansweredCount = computed(() =>
    this.unanswered.hasValue() ? this.unanswered.value().total : 0,
  );

  refreshReviews(): void {
    this.unanswered.reload();
  }
}
