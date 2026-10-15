import type { Person } from '@/core/api/types';
import { personName } from '@/core/lib/format';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

// A person's photo, or their initial on a brand gradient. `size` is the
// pixel size; extra classes (rings) go on the host.
@Component({
  selector: 'app-person-avatar',
  imports: [NgOptimizedImage],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex shrink-0 overflow-hidden rounded-full' },
  template: `
    @if (person().avatarUrl; as url) {
      <img [ngSrc]="url" [width]="size()" [height]="size()" [alt]="name()" class="rounded-full object-cover" />
    } @else {
      <span
        class="flex items-center justify-center rounded-full bg-linear-to-br from-brand-500 to-secondary-500 font-bold text-white"
        [style.width.px]="size()"
        [style.height.px]="size()"
        [style.font-size.px]="size() * 0.4"
        aria-hidden="true"
      >
        {{ initial() }}
      </span>
    }
  `,
})
export class PersonAvatar {
  readonly person = input.required<Person>();
  readonly size = input(40);
  protected readonly name = computed(() => personName(this.person()));
  protected readonly initial = computed(() => this.name().replace('@', '').charAt(0).toUpperCase());
}
