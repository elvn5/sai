export interface SupportInfo {
  id: string;
  message: string;
  phone: string | null;
  telegramContact: string | null;
  email: string | null;
  workingHours: string | null;
}

export interface AppUser {
  id: string;
  telegramId: string | null;
  email: string | null;
  username: string | null;
  firstName: string | null;
  lastName: string | null;
  phone: string | null;
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
