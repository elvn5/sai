import { DOCUMENT } from '@angular/common';
import { DestroyRef, type Signal, afterNextRender, inject, signal } from '@angular/core';

// How far below the top of the window a section counts as the one being
// read: under the sticky header and the tabs.
const OFFSET = 140;

export interface ScrollSpy<T extends string> {
  // The section being read.
  active: Signal<T>;
  // Scrolls to a section (smoothly, unless the visitor asked for less motion).
  go(id: T, event?: Event): void;
}

// Tabs that follow the page: the tab of the section under the header lights
// up while scrolling. Sections are found by id when the page scrolls, so
// they may render later (deferred blocks). Browser only.
export function injectScrollSpy<T extends string>(ids: readonly T[]): ScrollSpy<T> {
  const document = inject(DOCUMENT);
  const active = signal<T>(ids[0] as T);
  const destroyRef = inject(DestroyRef);

  afterNextRender(() => {
    let frame = 0;
    const update = (): void => {
      frame = 0;
      let current = ids[0] as T;
      for (const id of ids) {
        const element = document.getElementById(id);
        if (element && element.getBoundingClientRect().top <= OFFSET) current = id;
      }
      // At the very bottom the last section is the one being read.
      if (innerHeight + scrollY >= document.documentElement.scrollHeight - 2) {
        const last = ids[ids.length - 1] as T;
        if (document.getElementById(last)) current = last;
      }
      active.set(current);
    };
    const onScroll = (): void => {
      if (!frame) frame = requestAnimationFrame(update);
    };
    update();
    addEventListener('scroll', onScroll, { passive: true });
    destroyRef.onDestroy(() => {
      removeEventListener('scroll', onScroll);
      cancelAnimationFrame(frame);
    });
  });

  return {
    active: active.asReadonly(),
    go(id, event) {
      const element = document.getElementById(id);
      if (!element) return;
      event?.preventDefault();
      const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
      element.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' });
      active.set(id);
    },
  };
}
