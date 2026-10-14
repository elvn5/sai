// Shapes of the API (services/gateway) as the admin panel uses them.

export interface SupportInfo {
  id: string;
  message: string;
  phone: string | null;
  telegramContact: string | null;
  email: string | null;
  workingHours: string | null;
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
  createdAt: string;
  partnerStatus: PartnerStatus;
  partnerRequestedAt: string | null;
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
  // Set when a partner wrote it on the site; null for posts made here.
  userId: string | null;
  updatedAt: string;
}

export enum SeoPage {
  HOME = 'home',
  BLOG = 'blog',
  GALLERY = 'gallery',
  GUIDES = 'guides',
  TOURS = 'tours',
}

export const SEO_PAGE_OPTIONS: { value: SeoPage; label: string }[] = [
  { value: SeoPage.HOME, label: 'Главная' },
  { value: SeoPage.BLOG, label: 'Блог' },
  { value: SeoPage.GALLERY, label: 'Галерея' },
  { value: SeoPage.GUIDES, label: 'Гайды' },
  { value: SeoPage.TOURS, label: 'Туры' },
];

export interface PageSeo {
  id: string;
  page: SeoPage;
  locale: Locale;
  title: string | null;
  description: string | null;
  ogImageUrl: string | null;
}

export enum AchievementRarity {
  COMMON = 'common',
  RARE = 'rare',
  EPIC = 'epic',
  LEGENDARY = 'legendary',
}

// Texts keyed by locale; `ru` is required.
export type LocalizedText = Partial<Record<'ru' | 'ky' | 'en' | 'de', string>>;

export interface Achievement {
  id: string;
  title: LocalizedText;
  description: LocalizedText;
  icon: string;
  rarity: AchievementRarity;
  points: number;
  isActive: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface Person {
  id: string;
  firstName: string | null;
  username: string | null;
  avatarUrl: string | null;
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

export interface TrailSummary {
  id: string;
  slug: string;
  name: string;
  area: string;
  grade: string;
  isPublished: boolean;
  userId: string | null;
  createdAt: string;
}

export interface ConditionReportSummary {
  id: string;
  trail: { id: string; slug: string; name: string };
  author: Person;
  observedOn: string;
  trailState: string;
  traction: string;
  text: string | null;
  createdAt: string;
}

export type OrderStatus = 'pending' | 'paid' | 'failed' | 'expired' | 'refunded';

export interface AdminOrder {
  id: string;
  kind: string;
  itemTitle: string;
  itemDate: string | null;
  quantity: number;
  amount: number;
  fee: number;
  payeeAmount: number;
  contactName: string;
  contactPhone: string;
  status: OrderStatus;
  paidAt: string | null;
  refundNote: string | null;
  payoutId: string | null;
  createdAt: string;
  payer?: Person;
  payee?: Person;
}

export interface PayeeBalance {
  payee: Person;
  amount: number;
  orders: number;
}

export interface PayoutRecord {
  id: string;
  payee: Person;
  amount: number;
  orders: number;
  note: string | null;
  createdAt: string;
}
