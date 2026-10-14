import { type UseQueryReturnType, useQuery } from '@tanstack/vue-query';
import type { Ref } from 'vue';
import { useApiClient } from '~/shared/api/client';
import type {
  Order,
  Paginated,
  PartnerFinances,
  PartnerReview,
  StatsReport,
} from '~/shared/api/types';

// Data of the partner cabinet (/partner). Every query is the signed-in
// partner's own and waits for the session to say they are one.

export function usePartnerStats(days: Ref<number>): UseQueryReturnType<StatsReport, Error> {
  const api = useApiClient();
  const auth = useAuth();
  return useQuery({
    queryKey: ['partner', 'stats', days],
    queryFn: () => api.get<StatsReport>(`/listing-stats/mine?days=${days.value}`),
    enabled: auth.isPartner,
  });
}

export function usePartnerFinances(): UseQueryReturnType<PartnerFinances, Error> {
  const api = useApiClient();
  const auth = useAuth();
  return useQuery({
    queryKey: ['partner', 'finances'],
    queryFn: () => api.get<PartnerFinances>('/payouts/mine'),
    enabled: auth.isPartner,
  });
}

// Paid and refunded bookings of the partner's tours.
export function useReceivedOrders(): UseQueryReturnType<Order[], Error> {
  const api = useApiClient();
  const auth = useAuth();
  return useQuery({
    queryKey: ['orders', 'received'],
    queryFn: () => api.get<Order[]>('/orders/received'),
    enabled: auth.isPartner,
  });
}

export function usePartnerReviews(
  unanswered: Ref<boolean>,
  limit = 50,
): UseQueryReturnType<Paginated<PartnerReview>, Error> {
  const api = useApiClient();
  const auth = useAuth();
  return useQuery({
    queryKey: ['partner', 'reviews', unanswered, limit],
    queryFn: () =>
      api.get<Paginated<PartnerReview>>(
        `/reviews/mine?limit=${limit}${unanswered.value ? '&unanswered=true' : ''}`,
      ),
    enabled: auth.isPartner,
  });
}

// Change from `before` to `now` in whole percent; null when there's
// nothing to compare against.
export function percentChange(now: number, before: number): number | null {
  if (before === 0) return null;
  return Math.round(((now - before) / before) * 100);
}

// Share of `part` in `whole` as a percent string, e.g. "4.2%".
export function shareOf(part: number, whole: number): string {
  if (whole === 0) return '—';
  return `${(Math.round((part / whole) * 1000) / 10).toString()}%`;
}

// Today in Bishkek as YYYY-MM-DD: tour dates are Bishkek calendar days.
export function bishkekToday(): string {
  return new Date(Date.now() + 6 * 60 * 60 * 1000).toISOString().slice(0, 10);
}

// Downloads rows as a CSV file Excel opens correctly (BOM, semicolons).
export function downloadCsv(filename: string, rows: (string | number | null)[][]): void {
  const quoteCell = (cell: string | number | null): string => {
    const text = cell === null ? '' : String(cell);
    return /[";\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
  };
  const csv = `﻿${rows.map((row) => row.map(quoteCell).join(';')).join('\r\n')}`;
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
