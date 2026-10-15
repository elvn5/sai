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
