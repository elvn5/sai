import type {
  AdminTourInfo,
  ParticipantStatus,
  TourParticipant,
  TourPhase,
} from '@/core/api/types';
import { formatDate, personName } from '@/core/lib/format';

// Labels of the tours section, worded as on the site's partner pages.

export const PHASES: { value: TourPhase; label: string }[] = [
  { value: 'current', label: 'Сейчас идут' },
  { value: 'upcoming', label: 'Предстоящие' },
  { value: 'past', label: 'Прошедшие' },
];

const PHASE_LABELS: Record<TourPhase, string> = {
  current: 'идёт сейчас',
  upcoming: 'предстоит',
  past: 'прошёл',
};

export function phaseLabel(phase: TourPhase): string {
  return PHASE_LABELS[phase];
}

export function phaseTone(phase: TourPhase): string {
  if (phase === 'current') return 'bg-green-100 text-green-800';
  if (phase === 'upcoming') return 'bg-sky-100 text-sky-800';
  return 'bg-gray-100 text-gray-700';
}

export const PARTICIPANT_STATUSES: { value: ParticipantStatus; label: string }[] = [
  { value: 'new', label: 'Новая заявка' },
  { value: 'confirmed', label: 'Подтверждён' },
  { value: 'waitlist', label: 'Лист ожидания' },
  { value: 'cancelled', label: 'Отменил' },
];

export function statusLabel(status: ParticipantStatus): string {
  return PARTICIPANT_STATUSES.find((s) => s.value === status)?.label ?? status;
}

// "new" and "confirmed" take a place.
export function hasPlace(p: Pick<TourParticipant, 'status'>): boolean {
  return p.status === 'new' || p.status === 'confirmed';
}

const EXPERIENCE: Record<string, string> = {
  none: 'Нет опыта',
  hiking: 'Однодневные походы',
  trekking: 'Многодневные треки',
  mountaineering: 'Альпинизм',
};

export function experienceLabel(value: string): string {
  return EXPERIENCE[value] ?? value;
}

const EXPENSES: Record<string, string> = {
  transport: 'Транспорт',
  guides: 'Гиды',
  food: 'Питание',
  accommodation: 'Проживание',
  permits: 'Пропуска и сборы',
  gear: 'Снаряжение',
  other: 'Прочее',
};

export function expenseLabel(category: string): string {
  return EXPENSES[category] ?? category;
}

const ROLES: Record<string, string> = {
  guide: 'Гид',
  assistant: 'Помощник гида',
  driver: 'Водитель',
  cook: 'Повар',
  other: 'Другое',
};

export function roleLabel(role: string): string {
  return ROLES[role] ?? role;
}

const METHODS: Record<string, string> = {
  cash: 'Наличные',
  transfer: 'Перевод',
  card: 'Карта',
  other: 'Другое',
};

export function methodLabel(method: string): string {
  return METHODS[method] ?? method;
}

// "12.10.2026 — 14.10.2026", or one day.
export function tourDates(t: Pick<AdminTourInfo, 'startDate' | 'endDate'>): string {
  const start = formatDate(t.startDate);
  return t.endDate && t.endDate !== t.startDate ? `${start} — ${formatDate(t.endDate)}` : start;
}

// The group meets at this time (Bishkek); 09:00 when unset, as listings.
export function meetingTime(t: Pick<AdminTourInfo, 'meetingTime'>): string {
  return t.meetingTime || '09:00';
}

// "7 из 10", or "7" without a limit.
export function places(taken: number, capacity: number | null): string {
  return capacity ? `${taken} из ${capacity}` : `${taken}`;
}

// A participant's name: the account's, else what the organizer typed.
export function participantName(p: TourParticipant): string {
  if (p.account) return personName(p.account);
  return p.name ?? 'Без имени';
}

// A Telegram handle as a link: "@name" → https://t.me/name.
export function telegramLink(handle: string): string {
  return `https://t.me/${handle.replace(/^@/, '')}`;
}
