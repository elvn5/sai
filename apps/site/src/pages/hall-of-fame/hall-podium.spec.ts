import type { HallEntry } from '@/core/api/types';
import { HallPodium } from '@/pages/hall-of-fame/hall-podium';
import { pageTestProviders } from '@/testing/providers';
import { TestBed } from '@angular/core/testing';

function entry(rank: number): HallEntry {
  return {
    rank,
    points: 100 - rank,
    count: 1,
    achievements: [],
    user: { id: `u${rank}`, firstName: `Traveler ${rank}`, username: null, avatarUrl: null },
  };
}

describe('HallPodium', () => {
  it('keeps rank order for screen readers and lays the steps out 2 · 1 · 3', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(HallPodium);
    fixture.componentRef.setInput('leaders', [entry(1), entry(2), entry(3), entry(4)]);
    await fixture.whenStable();

    const steps = Array.from(fixture.nativeElement.querySelectorAll('li')) as HTMLElement[];
    expect(steps.map((step) => step.querySelector('p')?.textContent?.trim())).toEqual([
      'Traveler 1',
      'Traveler 2',
      'Traveler 3',
    ]);
    expect(steps.map((step) => step.style.order)).toEqual(['1', '0', '2']);
  });
});
