// Shapes of the API (services/gateway) as the site uses them.

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

// Who a post's trips suit; "any" for posts for every level.
export type BlogDifficulty = 'any' | 'easy' | 'moderate' | 'hard' | 'expert';
export type BlogSeason = 'spring' | 'summer' | 'autumn' | 'winter' | 'all_year';

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
  // Badges on the post's card, each optional.
  difficulty: BlogDifficulty | null;
  // Free text in the post's language ("2–5 часов").
  duration: string | null;
  season: BlogSeason | null;
  isPublished: boolean;
  publishedAt: string | null;
  // The partner who wrote it; null for posts from the editorial team.
  userId: string | null;
  updatedAt: string;
  // The locales the article is published in, its own included: posts under
  // the same slug are its translations. Only on GET /blog/:locale/:slug.
  translations?: string[];
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

// Whom a partner is about to award, found by their award code.
export interface AwardRecipient {
  user: Person;
  earnedAchievementIds: string[];
}

// The signed-in visitor's achievements and personal award code.
export interface MyAchievements {
  code: string;
  points: number;
  awards: Award[];
}

// Catalog listings (services/listings).

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

export enum HotelType {
  HOTEL = 'hotel',
  GUESTHOUSE = 'guesthouse',
  HOSTEL = 'hostel',
  YURT_CAMP = 'yurt_camp',
  RESORT = 'resort',
  APARTMENT = 'apartment',
}

export enum CancellationPolicy {
  FREE = 'free',
  FLEXIBLE = 'flexible',
  STRICT = 'strict',
  NON_REFUNDABLE = 'non_refundable',
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

export interface Tour extends ListingContacts {
  id: string;
  locale: string;
  slug: string;
  title: string;
  excerpt: string | null;
  content: string;
  price: number | null;
  durationDays: number | null;
  location: string | null;
  // Calendar dates (YYYY-MM-DD).
  startDate: string;
  endDate: string;
  destination: Destination | null;
  cancellationPolicy: CancellationPolicy | null;
  coverImageUrl: string | null;
  ratingAverage: number;
  reviewCount: number;
  isPublished: boolean;
  publishedAt: string | null;
  updatedAt: string;
  userId: string | null;
  photos: string[];
  program: TourDay[];
  // Places for travelers; null when unlimited.
  capacity: number | null;
  // Where and when (HH:MM, Bishkek time, on startDate) the group meets.
  meetingPoint: string | null;
  meetingTime: string | null;
  // Travelers signed up.
  taken: number;
  // The partner who runs it (absent for editorial tours).
  organizer?: PublicAuthor;
  // The signed-in visitor's: whether they signed up or run it (on
  // /tours/by-id/:id and their own lists; re-read in the browser).
  isJoined?: boolean;
  isOrganizer?: boolean;
  // The organizer's: who signed up and who came.
  roster?: TourParticipant[];
  // A participant's: whether they can rate it now, and their rating.
  canRate?: boolean;
  myRating?: { rating: number; text: string | null } | null;
}

// Someone signed up for a tour, as its organizer sees them.
export interface TourParticipant extends PublicAuthor {
  joinedAt: string;
  arrivedAt: string | null;
}

// One day of a tour's program.
export interface TourDay {
  title: string;
  text: string;
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

export interface ReviewReply {
  text: string;
  repliedAt: string;
}

export interface Review {
  id: string;
  rating: number;
  text: string;
  photos: string[];
  createdAt: string;
  author: PublicAuthor;
  // The listing owner's public answer.
  reply: ReviewReply | null;
}

export interface ReviewSummary {
  average: number;
  count: number;
  // Reviews per star, keyed "1"…"5".
  distribution: Record<string, number>;
}

export interface ReviewsPage extends Paginated<Review> {
  summary: ReviewSummary;
}

// News and photos a listing's owner posts on its page.
export interface ListingPost {
  id: string;
  text: string;
  photos: string[];
  createdAt: string;
  author: PublicAuthor;
}

export type ListingStatEvent = 'view' | 'phone' | 'whatsapp' | 'telegram' | 'website';

export type OrderStatus = 'pending' | 'paid' | 'failed' | 'expired' | 'refunded';

// A tour booking, paid on the payment provider's page (services/payments).
export interface Order {
  id: string;
  kind: 'tour';
  itemId: string;
  itemTitle: string;
  itemSlug: string | null;
  itemLocale: string | null;
  itemDate: string | null;
  // The tour's cover when booked.
  itemImageUrl: string | null;
  quantity: number;
  unitPrice: number;
  amount: number;
  fee: number;
  // What the partner gets.
  payeeAmount: number;
  contactName: string;
  contactPhone: string;
  comment: string | null;
  status: OrderStatus;
  paymentUrl: string | null;
  expiresAt: string;
  paidAt: string | null;
  refundNote: string | null;
  payoutId: string | null;
  createdAt: string;
}

// Partner cabinet

// One of the signed-in partner's listings (services/listings).
export interface ListingRef {
  targetType: ReviewTargetType;
  id: string;
  title: string;
  slug: string;
  // Tours only.
  locale: string | null;
  isPublished: boolean;
}

export interface PartnerReview extends Review {
  listing: ListingRef;
}

export interface StatCounts {
  views: number;
  phone: number;
  whatsapp: number;
  telegram: number;
  website: number;
  // All contact clicks together.
  contacts: number;
}

export interface DailyStats {
  day: string;
  views: number;
  contacts: number;
}

export interface ListingStatsRow extends StatCounts {
  listing: ListingRef;
}

export interface StatsReport {
  from: string;
  to: string;
  totals: StatCounts;
  // The same-length period before, for comparison.
  previous: StatCounts;
  daily: DailyStats[];
  listings: ListingStatsRow[];
}

export interface Payout {
  id: string;
  amount: number;
  orders: number;
  note: string | null;
  createdAt: string;
}

export interface PartnerFinances {
  // Paid bookings not yet paid out.
  owed: number;
  owedOrders: number;
  paidOut: number;
  month: { orders: number; guests: number; revenue: number; fee: number; earned: number };
  payouts: Payout[];
}

// services/trails

export interface TrailSegment {
  from: string;
  to: string;
  hours: number;
}

// [latitude, longitude, elevation | null]
export type TrackPoint = [number, number, number | null];

export interface TrackStats {
  distanceKm: number;
  elevationGain: number;
  minElevation: number | null;
  maxElevation: number | null;
}

export interface ConditionSummary {
  observedOn: string;
  trailState: string;
  traction: string;
}

export interface TrailMarker {
  pointIndex: number;
  label: string;
}

export interface Trail {
  markers?: TrailMarker[];
  checklistUrl?: string | null;
  id: string;
  slug: string;
  name: string;
  summary: string | null;
  description: string;
  area: string;
  kind: string;
  grade: string;
  seasons: string[];
  gear: string[];
  summitElevation: number | null;
  startElevation: number | null;
  elevationGain: number | null;
  distanceKm: number | null;
  durationHours: number | null;
  days: number;
  segments: TrailSegment[];
  gettingThere: string | null;
  startLatitude: number | null;
  startLongitude: number | null;
  // Only on the detail endpoint.
  track?: TrackPoint[];
  trackStats: TrackStats | null;
  gpxUrl: string | null;
  coverImageUrl: string | null;
  photos: string[];
  isPublished: boolean;
  userId: string | null;
  latestCondition: ConditionSummary | null;
  createdAt: string;
  updatedAt: string;
}

export interface TrailRef {
  id: string;
  slug: string;
  name: string;
  area: string;
}

// "What did the route look like?" — filed by anyone signed in after a trip.
export interface ConditionReport {
  id: string;
  trail: TrailRef;
  author: Person;
  observedOn: string;
  trailState: string;
  traction: string;
  avalancheRisk: string | null;
  snowLine: number | null;
  reached: string | null;
  text: string | null;
  photos: string[];
  createdAt: string;
}

// An organizer's public profile (users service).
export interface OrganizerProfile extends Person {
  // An approved partner.
  isVerified: boolean;
  memberSince: string;
}

export interface RatingSummary {
  average: number;
  count: number;
}

// An organizer's tours in a language (listings service).
export interface OrganizerTours {
  held: number;
  // Travelers who signed up for the tours held.
  participants: number;
  upcoming: Tour[];
  rating: RatingSummary;
  reviews: (Review & {
    listing: { id: string; title: string; slug: string; locale: string | null };
  })[];
}
