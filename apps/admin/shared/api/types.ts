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

export enum Locale {
  RU = 'ru',
  KY = 'ky',
  EN = 'en',
  DE = 'de',
}

export const LOCALE_OPTIONS: { value: Locale; label: string }[] = [
  { value: Locale.RU, label: 'Русский' },
  { value: Locale.KY, label: 'Кыргызча' },
  { value: Locale.EN, label: 'English' },
  { value: Locale.DE, label: 'Deutsch' },
];

export interface BlogPost {
  id: string;
  locale: Locale;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  coverImageUrl: string | null;
  seoTitle: string | null;
  seoDescription: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
}

export interface GalleryItem {
  id: string;
  imageUrl: string;
  caption: string | null;
  altText: string | null;
  sortOrder: number;
  isPublished: boolean;
}

export enum SeoPage {
  HOME = 'home',
  BLOG = 'blog',
  GALLERY = 'gallery',
}

export const SEO_PAGE_OPTIONS: { value: SeoPage; label: string }[] = [
  { value: SeoPage.HOME, label: 'Главная' },
  { value: SeoPage.BLOG, label: 'Блог' },
  { value: SeoPage.GALLERY, label: 'Галерея' },
];

export interface PageSeo {
  id: string;
  page: SeoPage;
  locale: Locale;
  title: string | null;
  description: string | null;
  ogImageUrl: string | null;
}

export enum GuideDifficulty {
  EASY = 'easy',
  MODERATE = 'moderate',
  HARD = 'hard',
}

export const GUIDE_DIFFICULTY_OPTIONS: { value: GuideDifficulty; label: string }[] = [
  { value: GuideDifficulty.EASY, label: 'Лёгкий' },
  { value: GuideDifficulty.MODERATE, label: 'Средний' },
  { value: GuideDifficulty.HARD, label: 'Сложный' },
];

export enum GuideSeason {
  SUMMER = 'summer',
  WINTER = 'winter',
  ALL_YEAR = 'all_year',
}

export const GUIDE_SEASON_OPTIONS: { value: GuideSeason; label: string }[] = [
  { value: GuideSeason.SUMMER, label: 'Лето' },
  { value: GuideSeason.WINTER, label: 'Зима' },
  { value: GuideSeason.ALL_YEAR, label: 'Круглый год' },
];

export interface Guide {
  id: string;
  locale: Locale;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  difficulty: GuideDifficulty;
  season: GuideSeason;
  equipmentChecklist: string[];
  coverImageUrl: string | null;
  seoTitle: string | null;
  seoDescription: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
}
