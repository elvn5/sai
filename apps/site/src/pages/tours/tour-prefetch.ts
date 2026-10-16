import type { Tour } from '@/core/api/types';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { PLATFORM_ID, inject } from '@angular/core';
import type { ResolveFn } from '@angular/router';
import { Router } from '@angular/router';
import { TranslocoService } from '@jsverse/transloco';
import { catchError, firstValueFrom, of, timeout } from 'rxjs';

// Don't hold a navigation longer than this for it.
const WAIT_MS = 1500;

// Fetches the tour before its page opens in the browser, so the page is
// whole when the view transition takes its picture and the card's cover can
// fly into the page's (motion.css). Not on the server or the first page
// load: those render with the page's own request.
export const prefetchTour: ResolveFn<Tour | null> = (route) => {
  if (!isPlatformBrowser(inject(PLATFORM_ID)) || !inject(Router).navigated) return null;
  const lang = inject(TranslocoService).getActiveLang();
  const slug = route.paramMap.get('slug') ?? '';
  return firstValueFrom(
    inject(HttpClient)
      .get<Tour>(`/tours/${lang}/${encodeURIComponent(slug)}`)
      .pipe(
        timeout(WAIT_MS),
        catchError(() => of(null)),
      ),
  );
};
