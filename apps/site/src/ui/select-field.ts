import { injectLang } from '@/core/i18n/current-lang';
import {
  CdkConnectedOverlay,
  CdkOverlayOrigin,
  type ConnectedPosition,
} from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  Injector,
  type Signal,
  afterNextRender,
  computed,
  inject,
  input,
  model,
  output,
  signal,
  viewChild,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

export interface SelectOption {
  value: string;
  label: string;
}

// Lists longer than this get a search box.
const SEARCH_FROM = 8;

// The list opens below the field, or above it when there's no room; it
// floats over the page (an overlay), so scrolling boxes and cards don't
// clip it.
const POSITIONS: ConnectedPosition[] = [
  { originX: 'start', originY: 'bottom', overlayX: 'start', overlayY: 'top', offsetY: 6 },
  { originX: 'end', originY: 'bottom', overlayX: 'end', overlayY: 'top', offsetY: 6 },
  { originX: 'start', originY: 'top', overlayX: 'start', overlayY: 'bottom', offsetY: -6 },
  { originX: 'end', originY: 'top', overlayX: 'end', overlayY: 'bottom', offsetY: -6 },
];

let nextId = 0;

// Search ignores case and «ё».
export function normalize(text: string): string {
  return text.toLocaleLowerCase().replaceAll('ё', 'е').trim();
}

// A select used everywhere instead of the browser's <select>: the field
// shows the chosen label, a click opens the list (with a search box for
// long lists). Keyboard: arrows, Home/End move, Enter picks, Escape
// closes; without a search box, typing a letter jumps to it. size "sm"
// fits table rows and cards. Works as a form control ([formField]) or with
// [value] / (valueChange).
@Component({
  selector: 'app-select-field',
  imports: [CdkConnectedOverlay, CdkOverlayOrigin, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'relative block',
    '(focusout)': 'onFocusOut($event)',
    '(keydown)': 'onKeydown($event)',
  },
  template: `
    <button
      #origin="cdkOverlayOrigin"
      cdkOverlayOrigin
      type="button"
      class="ui-input flex items-center gap-2 text-left"
      [class.ui-select-sm]="size() === 'sm'"
      aria-haspopup="listbox"
      [attr.aria-expanded]="open()"
      [attr.aria-controls]="listId"
      [attr.aria-labelledby]="labelledBy() ? labelledBy() + ' ' + triggerId : null"
      [attr.aria-invalid]="invalid() || null"
      [id]="triggerId"
      [disabled]="disabled()"
      (click)="toggle()"
    >
      <span class="min-w-0 flex-1 truncate" [class.text-text-muted]="!selected()">{{ selected()?.label ?? placeholder() }}</span>
      <span class="icon-[heroicons--chevron-up-down] size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
    </button>

    <ng-template
      cdkConnectedOverlay
      [cdkConnectedOverlayOrigin]="origin"
      [cdkConnectedOverlayOpen]="open()"
      [cdkConnectedOverlayPositions]="positions"
      [cdkConnectedOverlayMinWidth]="width()"
      [cdkConnectedOverlayPush]="true"
      (overlayOutsideClick)="onOutsideClick($event)"
      (detach)="close(false)"
    >
      <div #panel tabindex="-1" class="motion-drop-in w-max outline-none max-w-[min(22rem,calc(100vw-2rem))] min-w-full rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg" (keydown)="onKeydown($event)" (focusout)="onFocusOut($event)">
        @if (searchable()) {
          <label class="ui-input-icon m-1">
            <span class="icon-[heroicons--magnifying-glass]" aria-hidden="true"></span>
            <input
              class="ui-input"
              type="search"
              role="combobox"
              autocomplete="off"
              aria-autocomplete="list"
              [attr.aria-controls]="listId"
              aria-expanded="true"
              [attr.aria-activedescendant]="activeOptionId()"
              [attr.aria-label]="'picker.search' | transloco"
              [placeholder]="'picker.search' | transloco"
              [value]="query()"
              (input)="onSearch($event)"
            />
          </label>
        }
        <ul
          role="listbox"
          class="max-h-72 overflow-y-auto overscroll-contain p-0.5 outline-none"
          [id]="listId"
          [attr.aria-labelledby]="labelledBy()"
          [attr.aria-activedescendant]="searchable() ? null : activeOptionId()"
          [tabindex]="searchable() ? -1 : 0"
        >
          @for (option of filtered(); track option.value; let i = $index) {
            <li
              role="option"
              class="ui-option"
              [id]="listId + '-' + i"
              [attr.aria-selected]="option.value === value()"
              tabindex="-1"
              [attr.data-active]="i === active()"
              (mousemove)="active.set(i)"
              (click)="pick(option)"
              (keydown.enter)="pick(option)"
            >
              <span class="min-w-0 flex-1">{{ option.label }}</span>
              @if (option.value === value()) {
                <span class="icon-[heroicons--check] size-4 shrink-0 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
              }
            </li>
          } @empty {
            <li class="px-3 py-2 text-sm text-text-muted" role="presentation">{{ 'picker.noResults' | transloco }}</li>
          }
        </ul>
      </div>
    </ng-template>
  `,
})
export class SelectField implements FormValueControl<string> {
  readonly value = model('');
  readonly options = input.required<SelectOption[]>();
  // Shown when nothing is chosen (and no option has the empty value).
  readonly placeholder = input('');
  // A search box: "auto" from SEARCH_FROM options on.
  readonly search = input<boolean | 'auto'>('auto');
  // Id of the field's visible label.
  readonly labelledBy = input<string | null>(null);
  readonly disabled = input(false);
  readonly invalid = input(false);
  readonly size = input<'md' | 'sm'>('md');
  readonly touch = output<void>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly injector = inject(Injector);

  protected readonly triggerId = `select-${nextId}`;
  protected readonly listId = `select-list-${nextId++}`;
  protected readonly open = signal(false);
  protected readonly positions = POSITIONS;
  // The list is at least as wide as the field.
  protected readonly width = signal(0);
  private readonly panel = viewChild<ElementRef<HTMLElement>>('panel');
  protected readonly query = signal('');
  // Index of the highlighted option in the filtered list.
  protected readonly active = signal(0);

  protected readonly selected = computed(() =>
    this.options().find((option) => option.value === this.value()),
  );
  protected readonly searchable = computed(() =>
    this.search() === 'auto' ? this.options().length >= SEARCH_FROM : this.search() === true,
  );
  protected readonly filtered = computed(() => {
    const query = normalize(this.query());
    return query
      ? this.options().filter((option) => normalize(option.label).includes(query))
      : this.options();
  });
  protected readonly activeOptionId = computed(() =>
    this.filtered().length ? `${this.listId}-${this.active()}` : null,
  );

  protected toggle(): void {
    if (this.open()) this.close(false);
    else this.show();
  }

  private show(): void {
    this.query.set('');
    this.active.set(
      Math.max(
        0,
        this.options().findIndex((option) => option.value === this.value()),
      ),
    );
    this.width.set(this.trigger()?.offsetWidth ?? 0);
    this.open.set(true);
    afterNextRender(
      () => {
        const panel = this.panel()?.nativeElement;
        if (!panel) return;
        (
          panel.querySelector<HTMLElement>('input[type="search"]') ??
          panel.querySelector<HTMLElement>('[role="listbox"]')
        )?.focus();
        this.scrollActiveIntoView();
      },
      { injector: this.injector },
    );
  }

  private trigger(): HTMLElement | null {
    return this.host.nativeElement.querySelector<HTMLElement>(`#${this.triggerId}`);
  }

  protected close(returnFocus: boolean): void {
    if (!this.open()) return;
    this.open.set(false);
    this.touch.emit();
    if (returnFocus) this.trigger()?.focus();
  }

  protected pick(option: SelectOption): void {
    this.value.set(option.value);
    this.close(true);
  }

  protected onSearch(event: Event): void {
    this.query.set((event.target as HTMLInputElement).value);
    this.active.set(0);
  }

  protected onKeydown(event: KeyboardEvent): void {
    if (!this.open()) {
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        this.show();
      }
      return;
    }
    const count = this.filtered().length;
    const moves: Record<string, () => number> = {
      ArrowDown: () => Math.min(count - 1, this.active() + 1),
      ArrowUp: () => Math.max(0, this.active() - 1),
      Home: () => 0,
      End: () => count - 1,
    };
    if (event.key in moves) {
      if (!count) return;
      event.preventDefault();
      this.active.set(moves[event.key]());
      this.scrollActiveIntoView();
    } else if (event.key === 'Enter') {
      event.preventDefault();
      const option = this.filtered()[this.active()];
      if (option) this.pick(option);
    } else if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      this.close(true);
    } else if (event.key === 'Tab') {
      this.close(false);
    } else if (!this.searchable() && event.key.length === 1) {
      // Type-ahead: the next option starting with the letter.
      const letter = normalize(event.key);
      const options = this.filtered();
      for (let step = 1; step <= options.length; step++) {
        const i = (this.active() + step) % options.length;
        if (normalize(options[i].label).startsWith(letter)) {
          this.active.set(i);
          this.scrollActiveIntoView();
          break;
        }
      }
    }
  }

  // A click outside the list closes it; the field's own click toggles it.
  protected onOutsideClick(event: MouseEvent): void {
    if (!this.host.nativeElement.contains(event.target as Node)) this.close(false);
  }

  // Focus leaving both the field and the list closes it.
  protected onFocusOut(event: FocusEvent): void {
    const next = event.relatedTarget as Node | null;
    if (
      next &&
      !this.host.nativeElement.contains(next) &&
      !this.panel()?.nativeElement.contains(next)
    )
      this.close(false);
  }

  private scrollActiveIntoView(): void {
    afterNextRender(
      () => {
        const id = this.activeOptionId();
        const option = id
          ? (this.panel()?.nativeElement.querySelector<HTMLElement>(`#${id}`) ?? null)
          : null;
        const list = option?.parentElement;
        if (!option || !list) return;
        if (option.offsetTop < list.scrollTop) list.scrollTop = option.offsetTop;
        else if (option.offsetTop + option.offsetHeight > list.scrollTop + list.clientHeight)
          list.scrollTop = option.offsetTop + option.offsetHeight - list.clientHeight;
      },
      { injector: this.injector },
    );
  }
}

// Options whose labels are translation keys, translated again when the
// language changes (call in an injection context). `label` wins over `key`.
export function injectOptions(
  source: () => { value: string; key?: string; label?: string }[],
): Signal<SelectOption[]> {
  const transloco = inject(TranslocoService);
  const lang = injectLang();
  return computed(() => {
    lang();
    return source().map((o) => ({
      value: o.value,
      label: o.label ?? transloco.translate(o.key ?? o.value),
    }));
  });
}
