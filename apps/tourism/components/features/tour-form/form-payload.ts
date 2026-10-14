import type { CancellationPolicy, Destination } from '~/shared/api/types';

export interface TourFormPayload {
  title: string;
  excerpt: string | null;
  content: string;
  price: number | null;
  durationDays: number | null;
  location: string | null;
  startDate: string;
  endDate: string;
  destination: Destination | null;
  cancellationPolicy: CancellationPolicy;
  coverImageUrl: string | null;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
}
