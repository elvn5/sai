export interface LocalGuideFormPayload {
  name: string;
  headline: string | null;
  bio: string;
  avatarUrl: string | null;
  coverImageUrl: string | null;
  photos: string[];
  languages: string[];
  specializations: string[];
  destinations: string[];
  pricePerDay: number | null;
  experienceYears: number | null;
  hasVehicle: boolean;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
}
