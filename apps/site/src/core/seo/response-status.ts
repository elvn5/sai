import { RESPONSE_INIT, inject } from '@angular/core';

// Sets the HTTP status of the server-rendered page (404 for an unknown
// slug, 503 when the API failed: nginx doesn't cache 5xx, so an outage
// isn't kept for minutes). No-op in the browser. Call in an injection
// context; the returned function works anywhere.
export function injectResponseStatus(): (status: number) => void {
  const init = inject(RESPONSE_INIT, { optional: true });
  return (status) => {
    if (init) init.status = status;
  };
}
