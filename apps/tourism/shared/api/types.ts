export interface BlogPost {
  id: string;
  locale: string;
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

export enum GalleryRegion {
  BATKEN = 'batken',
  JALAL_ABAD = 'jalal_abad',
  ISSYK_KUL = 'issyk_kul',
  NARYN = 'naryn',
  OSH = 'osh',
  TALAS = 'talas',
  CHUY = 'chuy',
}

export interface GalleryItem {
  id: string;
  imageUrl: string;
  region: GalleryRegion;
  caption: string | null;
  altText: string | null;
  sortOrder: number;
  userId: string | null;
}

export enum GuideDifficulty {
  EASY = 'easy',
  MODERATE = 'moderate',
  HARD = 'hard',
}

export enum GuideSeason {
  SUMMER = 'summer',
  WINTER = 'winter',
  ALL_YEAR = 'all_year',
}

export interface GuideAttachment {
  url: string;
  name: string;
  size: number;
}

export interface Guide {
  id: string;
  locale: string;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  difficulty: GuideDifficulty;
  season: GuideSeason;
  equipmentChecklist: string[];
  coverImageUrl: string | null;
  attachments: GuideAttachment[];
  seoTitle: string | null;
  seoDescription: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
  userId: string | null;
}

export interface Tour {
  id: string;
  locale: string;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  price: number | null;
  durationDays: number | null;
  location: string | null;
  startDate: string;
  endDate: string;
  coverImageUrl: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
  userId: string | null;
}

export enum SeoPage {
  HOME = 'home',
  BLOG = 'blog',
  GALLERY = 'gallery',
  GUIDES = 'guides',
  TOURS = 'tours',
}

export interface PageSeo {
  id: string;
  page: SeoPage;
  locale: string;
  title: string | null;
  description: string | null;
  ogImageUrl: string | null;
}

export enum PartnerStatus {
  NONE = 'none',
  PENDING = 'pending',
  APPROVED = 'approved',
  REJECTED = 'rejected',
}

export interface AppUser {
  id: string;
  telegramId: string | null;
  email: string | null;
  username: string | null;
  firstName: string | null;
  lastName: string | null;
  phone: string | null;
  avatarUrl: string | null;
  partnerStatus: PartnerStatus;
}

export interface TelegramWidgetPayload {
  id: number;
  first_name: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
}
