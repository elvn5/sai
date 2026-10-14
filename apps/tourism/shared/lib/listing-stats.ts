import type { ListingStatEvent, ReviewTargetType } from '~/shared/api/types';

// Counts listing page views and contact clicks for the owner's partner
// cabinet. Fire-and-forget: keepalive lets a click that leaves for
// WhatsApp or a phone call still reach the server.
export function useListingStats(): {
  track: (targetType: ReviewTargetType, targetId: string, event: ListingStatEvent) => void;
} {
  const config = useRuntimeConfig();

  function track(targetType: ReviewTargetType, targetId: string, event: ListingStatEvent): void {
    if (import.meta.server) return;
    void fetch(`${config.public.apiBaseUrl}/listing-stats/events`, {
      method: 'POST',
      credentials: 'include',
      keepalive: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetType, targetId, event }),
    }).catch(() => undefined);
  }

  return { track };
}

// Counts a view of the listing page once it's shown in the browser.
export function useTrackListingView(
  targetType: ReviewTargetType,
  targetId: () => string | undefined,
): void {
  const { track } = useListingStats();
  onMounted(() => {
    const id = targetId();
    if (id) track(targetType, id, 'view');
  });
}
