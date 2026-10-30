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
  // A tour organizer or a driver who carries tour groups.
  partnerKind: PartnerKind;
  // Whether the account has a password (one made in Telegram may not).
  hasPassword?: boolean;
}

export type PartnerKind = 'organizer' | 'driver';

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
  // The visitor's place: 'new' or 'confirmed' (isJoined), 'waitlist' while
  // waiting for one; absent when they haven't signed up.
  myStatus?: ParticipantStatus;
  // The organizer's: who signed up and who came, and how many wait.
  roster?: TourParticipant[];
  waitlist?: number;
  // A participant's own form, and when they accepted the rules.
  myForm?: ParticipantForm;
  myWaiverAt?: string;
  // A participant's: whether they can rate it now, and their rating.
  canRate?: boolean;
  myRating?: { rating: number; text: string | null } | null;
}

export type MountainExperience = 'none' | 'hiking' | 'trekking' | 'mountaineering';

// What the organizer should know before the trip: whom to call in an
// emergency, mountain experience, health. Only the participant and the
// organizer see it.
export interface ParticipantForm {
  emergencyName: string;
  emergencyPhone: string;
  experience: MountainExperience;
  health: string | null;
  notes: string | null;
}

// A place on a tour: 'new' and 'confirmed' take one, 'waitlist' waits for
// one, 'cancelled' stays in the partner's history.
export type ParticipantStatus = 'new' | 'confirmed' | 'waitlist' | 'cancelled';

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

// --- The partner's CRM (/crm/*) ---

// One of the partner's clients: an account that signed up for their tours
// (account) or someone they added by hand. name, phone and telegram are
// what the partner typed in.
export interface CrmClient {
  id: string;
  userId: string | null;
  account: Person | null;
  name: string | null;
  phone: string | null;
  telegram: string | null;
  note: string | null;
  tags: string[];
  source: ClientSource | null;
  createdAt: string;
  // Places on the partner's tours (cancelled and waitlisted aside).
  stats: { tours: number; lastTour: string | null; due: number; paid: number };
  // On the card: their places, latest first, and the chat with them.
  history?: CrmClientTour[];
  chatId?: string;
  leads?: CrmLead[];
  tasks?: CrmTask[];
}

export interface CrmClientTour {
  participantId: string;
  tourId: string;
  title: string;
  slug: string;
  locale: string;
  startDate: string;
  status: ParticipantStatus;
  due: number;
  paid: number;
  arrived: boolean;
}

export type PaymentMethod = 'cash' | 'transfer' | 'card' | 'other';

export interface CrmPayment {
  id: string;
  // A refund counts against what they paid.
  kind: 'payment' | 'refund';
  amount: number;
  method: PaymentMethod;
  // YYYY-MM-DD
  paidOn: string;
  note: string | null;
  createdAt: string;
}

// One row of a tour's group.
export interface CrmParticipant {
  id: string;
  clientId: string;
  userId: string | null;
  account: Person | null;
  name: string | null;
  phone: string | null;
  telegram: string | null;
  status: ParticipantStatus;
  joinedAt: string;
  arrivedAt: string | null;
  // Set by the partner; null: the tour's price applies.
  amountDue: number | null;
  due: number;
  paid: number;
  payments: CrmPayment[];
  form: ParticipantForm | null;
  formAt: string | null;
  waiverAt: string | null;
  waiverVersion: string | null;
}

export interface CrmGroup {
  tour: Tour;
  participants: CrmParticipant[];
  totals: {
    active: number;
    waitlist: number;
    arrived: number;
    due: number;
    paid: number;
    // Of those with a place: forms filled, rules accepted.
    forms: number;
    waivers: number;
    expenses: number;
    // What's left of what was paid after the expenses.
    margin: number;
  };
  expenses: CrmExpense[];
  staff: CrmStaff[];
}

export interface CrmSummary {
  clients: number;
  paidThisMonth: number;
  outstanding: number;
  upcoming: {
    id: string;
    title: string;
    slug: string;
    locale: string;
    startDate: string;
    capacity: number | null;
    taken: number;
    waitlist: number;
    due: number;
    paid: number;
  }[];
}

// --- Chats between partners and travelers (/chats/*) ---

export interface ChatMessage {
  id: string;
  senderId: string;
  body: string;
  createdAt: string;
}

// One of the visitor's chats: 'partner' when they write to a client,
// 'client' when they write to an organizer.
export interface ChatSummary {
  id: string;
  partnerId: string;
  userId: string;
  role: 'partner' | 'client';
  peer: Person;
  lastMessage: ChatMessage;
  unread: number;
}

export interface ChatThread extends ChatSummary {
  peerReadAt: string | null;
  messages: ChatMessage[];
}

export type ClientSource =
  | 'ermekten'
  | 'instagram'
  | 'whatsapp'
  | 'telegram'
  | 'phone'
  | 'referral'
  | 'other';

export type ExpenseCategory =
  | 'transport'
  | 'guides'
  | 'food'
  | 'accommodation'
  | 'permits'
  | 'gear'
  | 'other';

export interface CrmExpense {
  id: string;
  category: ExpenseCategory;
  amount: number;
  note: string | null;
  createdAt: string;
}

export type StaffRole = 'guide' | 'assistant' | 'driver' | 'cook' | 'other';

export interface CrmStaff {
  id: string;
  role: StaffRole;
  name: string;
  phone: string | null;
  note: string | null;
}

// A client or a tour named on a lead or a task.
export interface CrmClientRef {
  id: string;
  userId: string | null;
  name: string | null;
  account: Person | null;
}

export interface CrmTourRef {
  id: string;
  title: string;
  startDate: string;
}

export type LeadStage = 'new' | 'working' | 'offer' | 'won' | 'lost';

export interface CrmLead {
  id: string;
  client: CrmClientRef;
  source: ClientSource | null;
  tour: CrmTourRef | null;
  stage: LeadStage;
  lostReason: string | null;
  note: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CrmTask {
  id: string;
  title: string;
  // YYYY-MM-DD
  dueOn: string;
  done: boolean;
  client: CrmClientRef | null;
  tour: CrmTourRef | null;
  doneAt: string | null;
}

export interface CrmTemplate {
  id: string;
  title: string;
  body: string;
}

export interface CrmBroadcast {
  id: string;
  // 'all', 'tour:<id>' or 'tag:<tag>'.
  audience: string;
  body: string;
  sent: number;
  skipped: number;
  createdAt: string;
}

export interface CrmReport {
  from: string;
  to: string;
  tours: number;
  places: number;
  taken: number;
  participants: number;
  arrived: number;
  noShows: number;
  cancelled: number;
  clients: number;
  repeatClients: number;
  revenue: number;
  expenses: number;
  leads: { source: ClientSource | null; total: number; won: number; lost: number }[];
  byTour: {
    id: string;
    title: string;
    startDate: string;
    capacity: number | null;
    taken: number;
    arrived: number;
    revenue: number;
    expenses: number;
  }[];
}

// --- The driver's CRM ---

export interface DriverProfile {
  userId: string;
  name: string;
  phone: string | null;
  region: string | null;
  about: string | null;
  published: boolean;
}

export type VehicleKind = 'sedan' | 'minivan' | 'minibus' | 'bus' | 'offroad' | 'other';

export interface DriverVehicle {
  id: string;
  kind: VehicleKind;
  model: string;
  plate: string | null;
  seats: number;
  year: number | null;
  // YYYY-MM-DD: when the insurance and the roadworthiness check run out.
  insuranceUntil: string | null;
  inspectionUntil: string | null;
  note: string | null;
  active: boolean;
}

export type TripStatus = 'requested' | 'accepted' | 'declined' | 'done' | 'cancelled';

export interface DriverTripTour {
  id: string;
  title: string;
  startDate: string;
  endDate: string;
  meetingPoint: string | null;
  meetingTime: string | null;
  participants: number;
}

export interface DriverTripClient {
  id: string;
  userId: string | null;
  account: Person | null;
  name: string | null;
  phone: string | null;
}

export interface DriverPayment {
  id: string;
  amount: number;
  method: PaymentMethod;
  paidOn: string;
  note: string | null;
  createdAt: string;
}

export type DriverExpenseCategory =
  | 'fuel'
  | 'repair'
  | 'maintenance'
  | 'wash'
  | 'parking'
  | 'fines'
  | 'insurance'
  | 'other';

export interface DriverExpense {
  id: string;
  category: DriverExpenseCategory;
  amount: number;
  spentOn: string;
  note: string | null;
  vehicleId: string | null;
  vehicle: string | null;
  tripId: string | null;
}

export interface DriverTrip {
  id: string;
  status: TripStatus;
  startsAt: string;
  endsAt: string | null;
  routeFrom: string;
  routeTo: string;
  passengers: number | null;
  price: number | null;
  paid: number;
  // The price less what was paid, for accepted and done trips.
  due: number;
  note: string | null;
  declineReason: string | null;
  vehicle: DriverVehicle | null;
  client: DriverTripClient | null;
  // Set when an organizer asked for the trip for one of their tours.
  tour: DriverTripTour | null;
  expenses: number;
  createdAt: string;
  payments?: DriverPayment[];
  expenseList?: DriverExpense[];
  chatId?: string | null;
}

export interface DriverDocument {
  vehicleId: string;
  vehicle: string;
  kind: 'insurance' | 'inspection';
  until: string;
}

export interface DriverSummary {
  requests: number;
  upcoming: DriverTrip[];
  monthIncome: number;
  monthExpenses: number;
  due: number;
  documents: DriverDocument[];
  hasProfile: boolean;
  published: boolean;
  vehicles: number;
}

export interface DriverReport {
  from: string;
  to: string;
  trips: number;
  cancelled: number;
  passengers: number;
  income: number;
  expenses: number;
  profit: number;
  byMonth: { month: string; trips: number; income: number; expenses: number }[];
  byCategory: Partial<Record<DriverExpenseCategory, number>>;
  byVehicle: {
    id: string | null;
    model: string | null;
    trips: number;
    income: number;
    expenses: number;
  }[];
}

// A driver in the organizers' catalog; busy on the day asked about.
export interface CatalogDriver extends DriverProfile {
  account: Person | null;
  vehicles: DriverVehicle[];
  trips: number;
  busy: boolean;
}

// Transport asked for a tour, on its group page.
export interface TransportRequest {
  id: string;
  status: TripStatus;
  startsAt: string;
  routeFrom: string;
  routeTo: string;
  passengers: number | null;
  price: number | null;
  note: string | null;
  declineReason: string | null;
  driver: DriverProfile;
  vehicle: DriverVehicle | null;
}

// The participant's form a traveler keeps in their profile
// (GET /tours/participant-form).
export interface ProfileParticipantForm {
  form: ParticipantForm | null;
  updatedAt: string | null;
}
