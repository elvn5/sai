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

export interface GalleryItem {
  id: string;
  imageUrl: string;
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
  seoTitle: string | null;
  seoDescription: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
  userId: string | null;
}

export interface PageSeo {
  id: string;
  page: string;
  locale: string;
  title: string | null;
  description: string | null;
  ogImageUrl: string | null;
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
