import { isPlatformBrowser } from '@angular/common';
import {
  DestroyRef,
  Directive,
  ElementRef,
  PLATFORM_ID,
  afterNextRender,
  computed,
  effect,
  inject,
  input,
  signal,
} from '@angular/core';

// Gives an element a view-transition-name, so it flies to the element of
// the same name on the next page (motion.css). In a list, only once its
// link (the closest <a>) is clicked: names must be unique on a page, and
// every named element costs the transition a picture.
@Directive({
  selector: '[appSharedTransition]',
  host: {
    class: 'vt-shared',
    '[style.view-transition-name]': 'name()',
  },
})
export class SharedTransition {
  readonly appSharedTransition = input.required<string>();
  // Named from the start (the page the card flies to).
  readonly sharedAlways = input(false);
  // Named only while this media query matches: a page with one cover for
  // phones and another for wide screens names the visible one.
  readonly sharedMedia = input<string | null>(null);

  private readonly clicked = signal(false);
  private readonly mediaMatches = signal(true);
  protected readonly name = computed(() =>
    (this.sharedAlways() || this.clicked()) && this.mediaMatches()
      ? this.appSharedTransition()
      : null,
  );

  constructor() {
    const element = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    const destroyRef = inject(DestroyRef);
    // Known before the first paint in the browser: two elements may never
    // carry the same name in one transition.
    const browser = isPlatformBrowser(inject(PLATFORM_ID));
    effect(() => {
      const media = this.sharedMedia();
      if (!browser || !media) return;
      const query = matchMedia(media);
      const update = (): void => this.mediaMatches.set(query.matches);
      update();
      query.addEventListener('change', update);
      destroyRef.onDestroy(() => query.removeEventListener('change', update));
    });
    afterNextRender(() => {
      const link = element.closest('a');
      if (!link) return;
      const onClick = (): void => this.clicked.set(true);
      link.addEventListener('click', onClick);
      destroyRef.onDestroy(() => link.removeEventListener('click', onClick));
    });
  }
}
