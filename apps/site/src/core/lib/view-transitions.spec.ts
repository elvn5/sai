import {
  guideAvatarTransition,
  hotelCoverTransition,
  tourCoverTransition,
  trailCoverTransition,
} from '@/core/lib/view-transitions';

describe('view-transitions', () => {
  it('names each kind of shared picture apart', () => {
    const names = [
      tourCoverTransition('1'),
      hotelCoverTransition('1'),
      trailCoverTransition('1'),
      guideAvatarTransition('1'),
    ];
    expect(new Set(names).size).toBe(4);
  });

  it('keeps only characters a view-transition-name allows', () => {
    expect(hotelCoverTransition('a1b2-c3d4_e5')).toBe('hotel-cover-a1b2-c3d4_e5');
    expect(trailCoverTransition('x/y z.1')).toBe('trail-cover-xyz1');
  });
});
