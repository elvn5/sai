export interface BannedItem {
  id: string;
  name: string;
  description: string | null;
  isActive: boolean;
  sortOrder: number;
}

export interface Address {
  id: string;
  title: string;
  fullAddress: string;
  comment: string | null;
  isActive: boolean;
  sortOrder: number;
}

export interface SupportInfo {
  id: string;
  message: string;
  phone: string | null;
  telegramContact: string | null;
  email: string | null;
  workingHours: string | null;
}

export enum ParcelStatus {
  CREATED = 'created',
  IN_TRANSIT = 'in_transit',
  ARRIVED = 'arrived',
  READY_FOR_PICKUP = 'ready_for_pickup',
  ISSUED = 'issued',
  CANCELLED = 'cancelled',
}

export const PARCEL_STATUS_OPTIONS: { value: ParcelStatus; label: string }[] = [
  { value: ParcelStatus.CREATED, label: 'Оформлена' },
  { value: ParcelStatus.IN_TRANSIT, label: 'В пути' },
  { value: ParcelStatus.ARRIVED, label: 'Прибыла на склад' },
  { value: ParcelStatus.READY_FOR_PICKUP, label: 'Готова к выдаче' },
  { value: ParcelStatus.ISSUED, label: 'Выдана' },
  { value: ParcelStatus.CANCELLED, label: 'Отменена' },
];

export interface AppUser {
  id: string;
  telegramId: string;
  username: string | null;
  firstName: string | null;
  lastName: string | null;
  phone: string | null;
  createdAt: string;
}

export interface Parcel {
  id: string;
  userId: string;
  user?: AppUser;
  trackNumber: string;
  description: string | null;
  status: ParcelStatus;
  weightKg: string | null;
  comment: string | null;
  createdAt: string;
}

export type PostLanguage = 'ru' | 'ky' | 'en';
export type PostStatus = 'draft' | 'published';

export interface Post {
  id: string;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  coverImageUrl: string | null;
  language: PostLanguage;
  status: PostStatus;
  publishedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export const POST_LANGUAGE_OPTIONS: { value: PostLanguage; label: string }[] = [
  { value: 'ru', label: 'Русский' },
  { value: 'ky', label: 'Кыргызча' },
  { value: 'en', label: 'English' },
];

export const POST_STATUS_OPTIONS: { value: PostStatus; label: string }[] = [
  { value: 'draft', label: 'Черновик' },
  { value: 'published', label: 'Опубликован' },
];
