import type {
  ClientSource,
  CrmClient,
  CrmGroup,
  CrmParticipant,
  LeadStage,
  MountainExperience,
  ParticipantStatus,
} from '@/core/api/types';
import type { Person } from '@/core/api/types';
import { personName } from '@/core/lib/format';
import type { CsvCell } from '@/core/lib/partner-cabinet';

export const CLIENT_SOURCES: ClientSource[] = [
  'ermekten',
  'instagram',
  'whatsapp',
  'telegram',
  'phone',
  'referral',
  'other',
];

export const LEAD_STAGES: LeadStage[] = ['new', 'working', 'offer', 'won', 'lost'];

// The order statuses are offered in.
export const PARTICIPANT_STATUSES: ParticipantStatus[] = [
  'new',
  'confirmed',
  'waitlist',
  'cancelled',
];

// The name a client goes by: the one the partner typed, else the account's.
export function clientName(client: Pick<CrmClient, 'name'> & { account: Person | null }): string {
  if (client.name) return client.name;
  return client.account ? personName(client.account) : '—';
}

export function participantName(p: CrmParticipant): string {
  return clientName(p);
}

// Badge class per status.
export function statusBadge(status: ParticipantStatus): string {
  switch (status) {
    case 'confirmed':
      return 'ui-badge ui-badge-success';
    case 'waitlist':
      return 'ui-badge ui-badge-warning';
    case 'cancelled':
      return 'ui-badge ui-badge-muted';
    default:
      return 'ui-badge ui-badge-info';
  }
}

// The group as CSV rows (header labels given), for a spreadsheet or the
// list for the border zone: everyone with a place, then the waitlist.
export function groupRows(
  group: CrmGroup,
  labels: {
    headers: string[];
    status: (s: ParticipantStatus) => string;
    experience: (e: MountainExperience) => string;
    yes: string;
  },
): CsvCell[][] {
  const rows = group.participants
    .filter((p) => p.status !== 'cancelled')
    .map((p, i): CsvCell[] => [
      i + 1,
      participantName(p),
      p.phone,
      p.telegram ? `@${p.telegram}` : p.account?.username ? `@${p.account.username}` : null,
      labels.status(p.status),
      p.arrivedAt ? labels.yes : '',
      p.due,
      p.paid,
      Math.max(p.due - p.paid, 0),
      p.form ? `${p.form.emergencyName} ${p.form.emergencyPhone}` : null,
      p.form ? labels.experience(p.form.experience) : null,
      p.form?.health ?? null,
      p.waiverAt ? labels.yes : '',
    ]);
  return [labels.headers, ...rows];
}
