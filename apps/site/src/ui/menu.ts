import { DOCUMENT, Directive, ElementRef, afterRenderEffect, inject, signal } from '@angular/core';

let nextId = 0;

// A small dropdown menu for the page frame (the header's account and
// language menus), without CDK Overlay: the frame is in every page's
// bundle, and these menus need no more than this. The panel sits in the
// same positioned box as the button:
//
//   <div appMenu #menu="appMenu" class="relative">
//     <button type="button" appMenuTrigger>…</button>
//     @if (menu.open()) {
//       <div appMenuPanel class="absolute right-0 top-full">
//         <a role="menuitem" …>…</a>
//       </div>
//     }
//   </div>
//
// Opens on click (or ArrowDown, focusing the first item); arrows, Home and
// End move between items; Escape, Tab, a click outside or on an item close
// it. Pages keep the CDK menus for richer ones (checkboxes, radios).
@Directive({
  selector: '[appMenu]',
  exportAs: 'appMenu',
  host: {
    '(document:click)': 'onDocumentClick($event)',
    '(keydown)': 'onKeydown($event)',
    '(click)': 'onClick($event)',
    '(focusout)': 'onFocusOut($event)',
  },
})
export class Menu {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly document = inject(DOCUMENT);
  readonly open = signal(false);
  readonly panelId = `menu-${nextId++}`;
  // Which item takes focus once the panel renders.
  private readonly focusOnOpen = signal<'first' | 'last' | null>(null);

  constructor() {
    afterRenderEffect(() => {
      const target = this.focusOnOpen();
      if (!this.open() || !target) return;
      const items = this.items();
      (target === 'first' ? items[0] : items.at(-1))?.focus();
      this.focusOnOpen.set(null);
    });
  }

  toggle(focus: 'first' | 'last' | null = null): void {
    if (this.open()) {
      this.close();
      return;
    }
    this.focusOnOpen.set(focus);
    this.open.set(true);
  }

  close(returnFocus = false): void {
    if (!this.open()) return;
    this.open.set(false);
    if (returnFocus) this.trigger()?.focus();
  }

  protected onDocumentClick(event: MouseEvent): void {
    if (!this.host.nativeElement.contains(event.target as Node)) this.close();
  }

  // A chosen item closes the menu (its own click handler runs first).
  protected onClick(event: MouseEvent): void {
    const item = (event.target as Element).closest('[role^="menuitem"]');
    if (item && this.host.nativeElement.contains(item)) this.close();
  }

  protected onFocusOut(event: FocusEvent): void {
    const next = event.relatedTarget as Node | null;
    if (next && !this.host.nativeElement.contains(next)) this.close();
  }

  protected onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape' && this.open()) {
      event.preventDefault();
      event.stopPropagation();
      this.close(true);
      return;
    }
    const items = this.items();
    const index = items.indexOf(this.document.activeElement as HTMLElement);
    if (index < 0) {
      // Open, with focus still on the button: the arrows step into the list.
      const edge = { ArrowDown: items[0], ArrowUp: items.at(-1) }[event.key];
      if (edge) {
        event.preventDefault();
        edge.focus();
      }
      return;
    }
    const next = {
      ArrowDown: items[(index + 1) % items.length],
      ArrowUp: items[(index - 1 + items.length) % items.length],
      Home: items[0],
      End: items.at(-1),
    }[event.key];
    if (next) {
      event.preventDefault();
      next.focus();
    } else if (event.key === 'Tab') {
      this.close();
    }
  }

  private items(): HTMLElement[] {
    return Array.from(
      this.host.nativeElement.querySelectorAll<HTMLElement>(`#${this.panelId} [role^="menuitem"]`),
    );
  }

  private trigger(): HTMLElement | null {
    return this.host.nativeElement.querySelector<HTMLElement>('[aria-haspopup="menu"]');
  }
}

// The button that opens the menu.
@Directive({
  selector: '[appMenuTrigger]',
  host: {
    'aria-haspopup': 'menu',
    '[attr.aria-expanded]': 'menu.open()',
    '[attr.aria-controls]': 'menu.open() ? menu.panelId : null',
    '(click)': 'menu.toggle()',
    '(keydown.arrowdown)': 'open($event, "first")',
    '(keydown.arrowup)': 'open($event, "last")',
  },
})
export class MenuTrigger {
  protected readonly menu = inject(Menu);

  protected open(event: Event, focus: 'first' | 'last'): void {
    event.preventDefault();
    if (!this.menu.open()) this.menu.toggle(focus);
  }
}

// The list of items; render it only while open (@if (menu.open())).
@Directive({
  selector: '[appMenuPanel]',
  host: { role: 'menu', '[id]': 'menu.panelId' },
})
export class MenuPanel {
  protected readonly menu = inject(Menu);
}
