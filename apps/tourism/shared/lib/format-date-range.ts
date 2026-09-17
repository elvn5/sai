export function formatDateRange(locale: string, startDate: string, endDate: string): string {
  const formatter = new Intl.DateTimeFormat(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
  if (startDate === endDate) return formatter.format(new Date(startDate));
  return `${formatter.format(new Date(startDate))} – ${formatter.format(new Date(endDate))}`;
}
