import { placesLeft } from '@/core/lib/places';

describe('placesLeft', () => {
  const tour = { id: 't1', capacity: 10 };

  it('subtracts the places bookings hold', () => {
    expect(placesLeft(tour, { t1: 7 })).toBe(3);
    expect(placesLeft(tour, {})).toBe(10);
  });

  it('never goes below zero', () => {
    expect(placesLeft(tour, { t1: 12 })).toBe(0);
  });

  it('is unknown without a limit or a count', () => {
    expect(placesLeft({ id: 't1', capacity: null }, { t1: 2 })).toBeNull();
    expect(placesLeft(tour, undefined)).toBeNull();
  });
});
