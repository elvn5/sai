import {
  daysUntil,
  fromLocalInput,
  toLocalInput,
  tripBadge,
  tripClientName,
  vehicleLabel,
} from '@/pages/partner/driver/driver';
import { describe, expect, it } from 'vitest';

describe('driver helpers', () => {
  it('reads and writes trip times in Bishkek time', () => {
    expect(fromLocalInput('2026-10-17T06:30')).toBe('2026-10-17T00:30:00.000Z');
    expect(toLocalInput('2026-10-17T00:30:00Z')).toBe('2026-10-17T06:30');
    expect(fromLocalInput('')).toBeNull();
  });

  it('counts the days until a document runs out', () => {
    expect(daysUntil('2026-10-29', '2026-11-28')).toBe(30);
    expect(daysUntil('2026-10-29', '2026-10-20')).toBe(-9);
  });

  it('names the customer and the vehicle', () => {
    expect(tripClientName({ client: null })).toBeNull();
    expect(
      tripClientName({
        client: {
          id: 'c',
          userId: 'u',
          name: null,
          phone: null,
          account: { id: 'u', firstName: 'Айгуль', username: null, avatarUrl: null },
        },
      }),
    ).toBe('Айгуль');
    expect(vehicleLabel({ model: 'Mercedes Sprinter', plate: null, seats: 18 })).toBe(
      'Mercedes Sprinter · 18',
    );
    expect(tripBadge('requested')).toContain('warning');
  });
});
