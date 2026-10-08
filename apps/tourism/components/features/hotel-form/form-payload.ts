import type { CancellationPolicy, Destination, HotelType } from '~/shared/api/types';

export interface HotelFormPayload {
  name: string;
  type: HotelType;
  description: string;
  destination: Destination;
  address: string | null;
  latitude: number | null;
  longitude: number | null;
  priceFrom: number | null;
  stars: number | null;
  amenities: string[];
  cancellationPolicy: CancellationPolicy;
  checkInTime: string | null;
  checkOutTime: string | null;
  coverImageUrl: string | null;
  photos: string[];
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
  website: string | null;
}
