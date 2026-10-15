import { ReviewTargetType } from '@/core/api/types';
import {
  contentLang,
  listingPath,
  percentChange,
  shareOf,
  toCsv,
} from '@/core/lib/partner-cabinet';

describe('partner cabinet', () => {
  it('compares periods in whole percent, and not against nothing', () => {
    expect(percentChange(150, 100)).toBe(50);
    expect(percentChange(90, 120)).toBe(-25);
    expect(percentChange(5, 0)).toBeNull();
  });

  it('shows a share with one decimal', () => {
    expect(shareOf(1, 24)).toBe('4.2%');
    expect(shareOf(3, 0)).toBe('—');
  });

  it('links listings to their public pages, tours in their own language', () => {
    const ref = { id: '1', title: 'x', slug: 'ala-kul', locale: 'en', isPublished: true };
    expect(listingPath({ ...ref, targetType: ReviewTargetType.TOUR })).toBe('/tours/ala-kul');
    expect(contentLang('en')).toBe('en');
    expect(contentLang(null)).toBeUndefined();
    expect(listingPath({ ...ref, targetType: ReviewTargetType.HOTEL })).toBe('/hotels/ala-kul');
  });

  it('writes CSV Excel reads: BOM, semicolons, quoted when needed', () => {
    const csv = toCsv([
      ['Тур', 'Гостей'],
      ['Ала-Куль; 3 дня', 2],
      ['Сон-Куль "выходные"', null],
    ]);
    expect(csv.startsWith('﻿')).toBe(true);
    expect(csv.slice(1).split('\r\n')).toEqual([
      'Тур;Гостей',
      '"Ала-Куль; 3 дня";2',
      '"Сон-Куль ""выходные""";',
    ]);
  });
});
