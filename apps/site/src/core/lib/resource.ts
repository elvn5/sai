import type { Resource } from '@angular/core';

// A resource's value, or undefined while it loads or after an error —
// unlike value(), which throws in the error state and would break the
// rendering of a whole page over one failed request.
export function loaded<T>(resource: Resource<T>): T | undefined {
  return resource.hasValue() ? resource.value() : undefined;
}
