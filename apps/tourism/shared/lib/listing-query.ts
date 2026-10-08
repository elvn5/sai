import type { LocationQuery } from 'vue-router';

// Helpers for keeping list filters in the URL, so links like
// /hotels?destination=karakol work and filtered views can be shared.

export function queryString(query: LocationQuery, key: string): string | undefined {
  const value = query[key];
  const first = Array.isArray(value) ? value[0] : value;
  return typeof first === 'string' && first.length > 0 ? first : undefined;
}

export function queryList(query: LocationQuery, key: string): string[] {
  return queryString(query, key)?.split(',').filter(Boolean) ?? [];
}

export function toQueryParams(filters: Record<string, unknown>): Record<string, string> {
  const params: Record<string, string> = {};
  for (const [key, value] of Object.entries(filters)) {
    if (value === undefined || value === null || value === '' || value === false) continue;
    if (Array.isArray(value)) {
      if (value.length > 0) params[key] = value.join(',');
      continue;
    }
    params[key] = String(value);
  }
  return params;
}
