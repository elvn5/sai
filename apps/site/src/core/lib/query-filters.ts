import type { QueryValue } from '@/core/lib/query-link';
import { inject } from '@angular/core';
import { Router } from '@angular/router';

// Writes filter changes into the URL (replacing the history entry) and
// starts over from page 1: the URL is the catalog's only state.
export function injectQueryFilters(): (changes: Record<string, QueryValue>) => void {
  const router = inject(Router);
  return (changes) => {
    const queryParams: Record<string, string | null> = { page: null };
    for (const [key, value] of Object.entries(changes)) {
      if (value === null || value === undefined || value === '' || value === false)
        queryParams[key] = null;
      else if (Array.isArray(value)) queryParams[key] = value.length > 0 ? value.join(',') : null;
      else queryParams[key] = String(value);
    }
    void router.navigate([], { queryParams, queryParamsHandling: 'merge', replaceUrl: true });
  };
}
