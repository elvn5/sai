import { isPlatformBrowser } from '@angular/common';
import {
  CUSTOM_ELEMENTS_SCHEMA,
  ChangeDetectionStrategy,
  Component,
  PLATFORM_ID,
  computed,
  inject,
  input,
} from '@angular/core';

let registered: Promise<unknown> | null = null;

// An icon named at runtime, as stored by the API ("i-mdi-trophy", the Nuxt
// Icon convention): achievement icons set in the admin. The Iconify web
// component draws it in the browser (fetched from the Iconify API on
// demand); the server leaves the slot empty. Static icons use the
// icon-[set--name] CSS classes instead.
@Component({
  selector: 'app-icon',
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex', 'aria-hidden': 'true' },
  template: '<iconify-icon [attr.icon]="iconifyName()" width="100%" height="100%" />',
})
export class Icon {
  readonly name = input.required<string>();

  // "i-mdi-map-marker-path" → "mdi:map-marker-path"
  protected readonly iconifyName = computed(() => {
    const raw = this.name().replace(/^i-/, '');
    const dash = raw.indexOf('-');
    return dash > 0 ? `${raw.slice(0, dash)}:${raw.slice(dash + 1)}` : raw;
  });

  constructor() {
    // The web component touches HTMLElement at import: browser only.
    if (isPlatformBrowser(inject(PLATFORM_ID))) registered ??= import('iconify-icon');
  }
}
