import type { Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { TourBooking } from '@/pages/tours/tour-booking';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';

const tour = {
  price: 12000,
  userId: 'partner',
  startDate: '2999-01-01',
  endDate: '2999-01-03',
} as Tour;

async function render(online: boolean): Promise<HTMLElement> {
  TestBed.configureTestingModule({
    providers: [...pageTestProviders(), { provide: AuthStore, useValue: { user: signal(null) } }],
  });
  const fixture = TestBed.createComponent(TourBooking);
  fixture.componentRef.setInput('tour', tour);
  TestBed.tick();
  TestBed.inject(HttpTestingController).expectOne('/payments/config').flush({ online });
  await fixture.whenStable();
  return fixture.nativeElement as HTMLElement;
}

describe('TourBooking', () => {
  it('offers booking while online payment is on', async () => {
    expect((await render(true)).querySelector('section')).not.toBeNull();
  });

  it('hides booking when online payment is off', async () => {
    expect((await render(false)).querySelector('section')).toBeNull();
  });
});
