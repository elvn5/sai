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
  updatedAt: string;
  partnerStatus: PartnerStatus;
  partnerRequestedAt: string | null;
  // A tour organizer or a driver who carries tour groups.
  partnerKind: 'organizer' | 'driver';
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

// Who a post's trips suit; "any" for posts for every level.
export type BlogDifficulty = 'any' | 'easy' | 'moderate' | 'hard' | 'expert';
export type BlogSeason = 'spring' | 'summer' | 'autumn' | 'winter' | 'all_year';

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
  // Badges on the post's card, each optional.
  difficulty: BlogDifficulty | null;
  // Free text in the post's language ("2–5 часов").
  duration: string | null;
  season: BlogSeason | null;
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

// GoPay's own statuses of a payment.
export type TransactionStatus =
  | 'CREATED'
  | 'PENDING'
  | 'COMMITTED'
  | 'FAILED'
  | 'EXPIRED'
  | 'CANCELLED';

// A webhook delivery from the gateway about a payment.
export interface ProviderEvent {
  id: number;
  event: string;
  paymentId: string | null;
  testing: boolean;
  attempts: number;
  receivedAt: string;
  processedAt: string | null;
  error: string | null;
}

// The gateway's record of an order's payment (GET /payments/admin/transactions).
export interface Transaction {
  orderId: string;
  provider: string;
  paymentId: string | null;
  providerOrderId: string;
  status: TransactionStatus;
  amount: string;
  testing: boolean;
  checkoutUrl: string | null;
  gatewayCreatedAt: string | null;
  expiresAt: string | null;
  committedAt: string | null;
  bankOpDate: string | null;
  lastError: string | null;
  checkedAt: string;
  createdAt: string;
  updatedAt: string;
  order: AdminOrder;
  events?: ProviderEvent[];
}

export interface PaymentGateway {
  provider: string;
  online: boolean;
  canCancel: boolean;
}

// Tours (listings, GET /tours/admin): where a departure stands — before
// the group meets, on the way (until the evening of its last day), over.
export type TourPhase = 'upcoming' | 'current' | 'past';

export interface AdminTourInfo {
  id: string;
  locale: string;
  slug: string;
  title: string;
  location: string | null;
  price: number | null;
  startDate: string;
  endDate: string;
  // HH:MM in Bishkek on startDate; 09:00 when unset.
  meetingTime: string | null;
  meetingPoint: string | null;
  capacity: number | null;
  // Places taken.
  taken: number;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
  isPublished: boolean;
  publishedAt: string | null;
  userId: string | null;
  organizer?: Person;
  createdAt: string;
}

// A tour's participants by status; of those with a place: who came,
// filled the form, accepted the rules, what they owe and paid.
export interface AdminTourCounts {
  active: number;
  waitlist: number;
  cancelled: number;
  arrived: number;
  forms: number;
  waivers: number;
  due: number;
  paid: number;
}

export interface AdminTourRow {
  tour: AdminTourInfo;
  phase: TourPhase;
  counts: AdminTourCounts;
}

export interface AdminTourList {
  tours: AdminTourRow[];
  // Tours per phase (after the search).
  counts: Record<TourPhase, number>;
}

// "new" and "confirmed" take a place; "waitlist" waits for one.
export type ParticipantStatus = 'new' | 'confirmed' | 'waitlist' | 'cancelled';

export interface ParticipantForm {
  emergencyName: string;
  emergencyPhone: string;
  // none, hiking, trekking or mountaineering.
  experience: string;
  health: string | null;
  notes: string | null;
}

export interface TourPayment {
  id: string;
  // A refund counts against what they paid.
  kind: 'payment' | 'refund';
  amount: number;
  method: string;
  paidOn: string;
  note: string | null;
  createdAt: string;
}

export interface TourParticipant {
  id: string;
  // An account that signed up, or a client the organizer added by hand.
  userId: string | null;
  account: Person | null;
  // What the organizer typed for the client.
  name: string | null;
  phone: string | null;
  telegram: string | null;
  status: ParticipantStatus;
  joinedAt: string;
  arrivedAt: string | null;
  amountDue: number | null;
  due: number;
  paid: number;
  payments: TourPayment[];
  form: ParticipantForm | null;
  formAt: string | null;
  waiverAt: string | null;
  waiverVersion: string | null;
}

export interface TourTotals {
  active: number;
  waitlist: number;
  arrived: number;
  due: number;
  paid: number;
  forms: number;
  waivers: number;
  expenses: number;
  margin: number;
}

export interface TourExpense {
  id: string;
  category: string;
  amount: number;
  note: string | null;
  createdAt: string;
}

export interface TourStaff {
  id: string;
  role: string;
  name: string;
  phone: string | null;
  note: string | null;
}

// One tour's group (GET /tours/admin/:id).
export interface AdminTourGroup {
  tour: AdminTourInfo;
  phase: TourPhase;
  organizer: Person | null;
  participants: TourParticipant[];
  totals: TourTotals;
  expenses: TourExpense[] | null;
  staff: TourStaff[] | null;
}
