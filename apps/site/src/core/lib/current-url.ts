import { type Signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { filter, map } from 'rxjs';

// The URL of the page shown (after each finished navigation), for
// components outside the router outlet such as the header.
export function injectCurrentUrl(): Signal<string> {
  const router = inject(Router);
  return toSignal(
    router.events.pipe(
      filter((event) => event instanceof NavigationEnd),
      map((event) => event.urlAfterRedirects),
    ),
    { initialValue: router.url },
  );
}
