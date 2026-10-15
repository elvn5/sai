import { type ListingRef, ReviewTargetType } from '@/core/api/types';
import { LANGS, type Lang } from '@/core/i18n/langs';

// Helpers of the partner cabinet (/partner) and of bookings.

// Change from `before` to `now` in whole percent; null when there's
// nothing to compare against.
export function percentChange(now: number, before: number): number | null {
  if (before === 0) return null;
  return Math.round(((now - before) / before) * 100);
}

// Share of `part` in `whole` as a percent string, e.g. "4.2%".
export function shareOf(part: number, whole: number): string {
  if (whole === 0) return '—';
  return `${Math.round((part / whole) * 1000) / 10}%`;
}

// The language a tour is written in: its page lives only under that
// language ("/en/tours/…"), whatever language the visitor reads. For
// SiteLink's [lang]; undefined keeps the current one.
export function contentLang(locale: string | null | undefined): Lang | undefined {
  return LANGS.find((lang) => lang === locale);
}

// The unprefixed public path of one of a partner's listings (with
// contentLang(listing.locale) for tours).
export function listingPath(listing: ListingRef): string {
  switch (listing.targetType) {
    case ReviewTargetType.HOTEL:
      return `/hotels/${listing.slug}`;
    case ReviewTargetType.LOCAL_GUIDE:
      return `/local-guides/${listing.slug}`;
    case ReviewTargetType.TOUR:
      return `/tours/${listing.slug}`;
  }
}

export type CsvCell = string | number | null;

// CSV that Excel opens correctly: a BOM, semicolons, quoted when needed.
export function toCsv(rows: CsvCell[][]): string {
  const cell = (value: CsvCell): string => {
    const text = value === null ? '' : String(value);
    return /[";\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
  };
  return `﻿${rows.map((row) => row.map(cell).join(';')).join('\r\n')}`;
}

// Saves rows as a CSV file (browser only).
export function downloadCsv(document: Document, filename: string, rows: CsvCell[][]): void {
  const url = URL.createObjectURL(new Blob([toCsv(rows)], { type: 'text/csv;charset=utf-8' }));
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
