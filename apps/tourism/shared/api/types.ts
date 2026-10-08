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
  location: string | null;
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
  destination: Destination | null;
  cancellationPolicy: CancellationPolicy | null;
  coverImageUrl: string | null;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
  ratingAverage: number;
  reviewCount: number;
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

export enum Destination {
  ISSYK_KUL = 'issyk_kul',
  KARAKOL = 'karakol',
  BISHKEK = 'bishkek',
  ALA_ARCHA = 'ala_archa',
  SONG_KOL = 'song_kol',
  NARYN = 'naryn',
  TASH_RABAT = 'tash_rabat',
  KEL_SUU = 'kel_suu',
  OSH = 'osh',
  ARSLANBOB = 'arslanbob',
  SARY_CHELEK = 'sary_chelek',
  TALAS = 'talas',
  BATKEN = 'batken',
  CHUY = 'chuy',
}

export enum CancellationPolicy {
  FREE = 'free',
  FLEXIBLE = 'flexible',
  STRICT = 'strict',
  NON_REFUNDABLE = 'non_refundable',
}

export enum HotelType {
  HOTEL = 'hotel',
  GUESTHOUSE = 'guesthouse',
  HOSTEL = 'hostel',
  YURT_CAMP = 'yurt_camp',
  RESORT = 'resort',
  APARTMENT = 'apartment',
}

export enum ReviewTargetType {
  HOTEL = 'hotel',
  LOCAL_GUIDE = 'local_guide',
  TOUR = 'tour',
}

export interface ListingContacts {
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
}

export interface Hotel extends ListingContacts {
  id: string;
  slug: string;
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
  website: string | null;
  ratingAverage: number;
  reviewCount: number;
  userId: string | null;
  createdAt: string;
}

export interface LocalGuide extends ListingContacts {
  id: string;
  slug: string;
  name: string;
  headline: string | null;
  bio: string;
  avatarUrl: string | null;
  coverImageUrl: string | null;
  photos: string[];
  languages: string[];
  specializations: string[];
  destinations: Destination[];
  pricePerDay: number | null;
  experienceYears: number | null;
  hasVehicle: boolean;
  ratingAverage: number;
  reviewCount: number;
  userId: string | null;
  createdAt: string;
}

export interface Paginated<T> {
  items: T[];
  total: number;
}

export interface PublicAuthor {
  id: string;
  firstName: string | null;
  username: string | null;
  avatarUrl: string | null;
}

export interface Review {
  id: string;
  rating: number;
  text: string;
  photos: string[];
  createdAt: string;
  author: PublicAuthor;
}

export interface ReviewSummary {
  average: number;
  count: number;
  distribution: Record<string, number>;
}

export interface ReviewsPage extends Paginated<Review> {
  summary: ReviewSummary;
}

export interface ListingPost {
  id: string;
  text: string;
  photos: string[];
  createdAt: string;
  author: PublicAuthor;
}
