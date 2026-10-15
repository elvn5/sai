export type QueryValue = string | number | boolean | readonly string[] | null | undefined;

// "/hotels?destination=karakol&page=2" from a path and values, leaving out
// empty ones (blank, false, [], page 1) so equal states give equal links.
export function withQuery(path: string, values: Record<string, QueryValue>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value === null || value === undefined || value === '' || value === false) continue;
    if (key === 'page' && Number(value) <= 1) continue;
    if (Array.isArray(value)) {
      if (value.length > 0) params.set(key, value.join(','));
      continue;
    }
    params.set(key, String(value));
  }
  const query = params.toString();
  return query ? `${path}?${query}` : path;
}

// A positive page number from a ?page= value; 1 for anything else.
export function pageNumber(value: string | undefined): number {
  return Math.max(1, Number.parseInt(value ?? '1', 10) || 1);
}

// A comma-separated list from a query value ("wifi,sauna").
export function queryList(value: string | undefined): string[] {
  return value?.split(',').filter(Boolean) ?? [];
}
