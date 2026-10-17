import {
  DestroyRef,
  Directive,
  ElementRef,
  afterNextRender,
  computed,
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

  private readonly clicked = signal(false);
  protected readonly name = computed(() =>
    this.sharedAlways() || this.clicked() ? this.appSharedTransition() : null,
  );

  constructor() {
    const element = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const link = element.closest('a');
      if (!link) return;
      const onClick = (): void => this.clicked.set(true);
      link.addEventListener('click', onClick);
      destroyRef.onDestroy(() => link.removeEventListener('click', onClick));
    });
  }
}
