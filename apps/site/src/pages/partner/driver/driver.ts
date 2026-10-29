import type {
  DriverExpenseCategory,
  DriverTrip,
  DriverVehicle,
  TripStatus,
  VehicleKind,
} from '@/core/api/types';
import { personName } from '@/core/lib/format';

export const VEHICLE_KINDS: VehicleKind[] = [
  'minivan',
  'minibus',
  'bus',
  'offroad',
  'sedan',
  'other',
];

export const TRIP_STATUSES: TripStatus[] = [
  'requested',
  'accepted',
  'done',
  'declined',
  'cancelled',
];

export const DRIVER_EXPENSE_CATEGORIES: DriverExpenseCategory[] = [
  'fuel',
  'repair',
  'maintenance',
  'wash',
  'parking',
  'fines',
  'insurance',
  'other',
];

// Kyrgyzstan keeps UTC+6 all year: trips are typed in and shown in it.
const BISHKEK_OFFSET = '+06:00';

// An instant as the value of a datetime-local field, in Bishkek.
export function toLocalInput(iso: string): string {
  const shifted = new Date(new Date(iso).getTime() + 6 * 3600_000);
  return shifted.toISOString().slice(0, 16);
}

// A datetime-local value (Bishkek time) as an instant; '' stays null.
export function fromLocalInput(value: string): string | null {
  return value ? new Date(`${value}:00${BISHKEK_OFFSET}`).toISOString() : null;
}

// Badge class per trip status.
export function tripBadge(status: TripStatus): string {
  switch (status) {
    case 'requested':
      return 'ui-badge ui-badge-warning';
    case 'accepted':
      return 'ui-badge ui-badge-info';
    case 'done':
      return 'ui-badge ui-badge-success';
    default:
      return 'ui-badge ui-badge-muted';
  }
}

// Who ordered a trip: the name the driver typed, else the account's.
export function tripClientName(trip: Pick<DriverTrip, 'client'>): string | null {
  const client = trip.client;
  if (!client) return null;
  if (client.name) return client.name;
  return client.account ? personName(client.account) : null;
}

// A vehicle in one line: "Mercedes Sprinter · 01KG123ABC · 18".
export function vehicleLabel(v: Pick<DriverVehicle, 'model' | 'plate' | 'seats'>): string {
  return [v.model, v.plate, `${v.seats}`].filter(Boolean).join(' · ');
}

// Days from today (YYYY-MM-DD) until a date; negative once it's passed.
export function daysUntil(today: string, date: string): number {
  return Math.round(
    (Date.parse(`${date}T00:00:00Z`) - Date.parse(`${today}T00:00:00Z`)) / 86_400_000,
  );
}
