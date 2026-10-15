import 'iconify-icon';
import {
  CUSTOM_ELEMENTS_SCHEMA,
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
} from '@angular/core';

// An icon named at runtime, as stored by the API ("i-mdi-trophy", the
// Nuxt Icon convention). Static icons use the icon-[set--name] CSS classes
// instead; these come from the Iconify API on demand.
@Component({
  selector: 'app-icon',
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex' },
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
}
