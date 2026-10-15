// Shapes of the API (services/gateway) as the site uses them, from
// apps/tourism/shared/api/types.ts; types move here as their pages do.

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

// What the Telegram Login Widget hands to its callback.
export interface TelegramWidgetPayload {
  id: number;
  first_name: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
}

// Pages whose title, description and preview image the admin can override.
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
  // The partner who wrote it; null for posts from the editorial team.
  userId: string | null;
  updatedAt: string;
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

export interface GalleryItem {
  id: string;
  imageUrl: string;
  location: string | null;
  caption: string | null;
  altText: string | null;
  sortOrder: number;
  userId: string | null;
}

// Someone shown on public pages.
export interface Person {
  id: string;
  firstName: string | null;
  username: string | null;
  avatarUrl: string | null;
}

export type LocalizedText = Partial<Record<'ru' | 'ky' | 'en' | 'de', string>>;

export enum AchievementRarity {
  COMMON = 'common',
  RARE = 'rare',
  EPIC = 'epic',
  LEGENDARY = 'legendary',
}

export interface Achievement {
  id: string;
  title: LocalizedText;
  description: LocalizedText;
  // Iconify name in the Nuxt Icon form ("i-mdi-trophy"), set in the admin.
  icon: string;
  rarity: AchievementRarity;
  points: number;
  isActive: boolean;
  sortOrder: number;
}

export interface Award {
  id: string;
  achievement: Achievement;
  user: Person;
  awardedBy: Person | null;
  note: string | null;
  awardedAt: string;
  isNew: boolean;
}

export interface HallEntry {
  rank: number;
  user: Person;
  points: number;
  count: number;
  achievements: Achievement[];
}

export interface HallOfFame {
  leaders: HallEntry[];
  recent: Award[];
}

// The signed-in visitor's achievements and personal award code.
export interface MyAchievements {
  code: string;
  points: number;
  awards: Award[];
}
