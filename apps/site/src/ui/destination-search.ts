import { DESTINATION_OPTIONS } from '@/core/lib/listing-options';
import { withQuery } from '@/core/lib/query-link';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  computed,
  inject,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

let nextId = 0;

// "Where to?": a combobox over the destinations that opens their tours
// (/tours?destination=…). Arrows move, Enter opens, Escape closes. The
// hero's version is a big white bar with a button; the header's is a field.
@Component({
  selector: 'app-destination-search',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'relative block' },
  template: `
    <form
      role="search"
      [class]="
        variant() === 'hero'
          ? 'flex gap-1.5 rounded-[0.875rem] bg-white p-1.5 text-slate-900 shadow-[0_20px_40px_-12px_rgb(0_0_0/0.45)]'
          : 'flex'
      "
      (submit)="submit($event)"
    >
      <label class="relative flex min-w-0 flex-1 items-center">
        <span class="sr-only">{{ 'search.label' | transloco }}</span>
        <span
          class="icon-[heroicons--magnifying-glass] pointer-events-none absolute left-3 size-[1.1rem]"
          [class]="variant() === 'hero' ? 'text-slate-500' : 'text-text-muted'"
          aria-hidden="true"
        ></span>
        <input
          #field
          type="text"
          role="combobox"
          autocomplete="off"
          [attr.aria-expanded]="open()"
          [attr.aria-controls]="listId"
          [attr.aria-activedescendant]="active() >= 0 ? listId + '-' + active() : null"
          aria-autocomplete="list"
          [placeholder]="(variant() === 'hero' ? 'search.placeholder' : 'search.placeholderShort') | transloco"
          [value]="query()"
          [class]="
            variant() === 'hero'
              ? 'h-11 w-full rounded-lg bg-transparent pr-2 pl-9 text-[0.9375rem] text-slate-900 outline-none placeholder:text-slate-500'
              : 'ui-input pl-9'
          "
          (input)="onInput($any($event.target).value)"
          (focus)="open.set(true)"
          (blur)="close()"
          (keydown)="onKeydown($event)"
        />
      </label>
      @if (variant() === 'hero') {
        <button type="submit" class="ui-btn-primary ui-btn-lg shrink-0 max-sm:hidden">{{ 'search.submit' | transloco }}</button>
      }
    </form>
    @if (open() && matches().length > 0) {
      <ul
        [id]="listId"
        role="listbox"
        [attr.aria-label]="'search.label' | transloco"
        class="motion-drop-in absolute inset-x-0 top-full z-50 mt-2 max-h-72 overflow-y-auto rounded-xl border border-border bg-surface p-1 text-left text-sm text-text shadow-lg"
      >
        @for (option of matches(); track option.value; let index = $index) {
          <li
            [id]="listId + '-' + index"
            role="option"
            [attr.aria-selected]="index === active()"
            class="flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 aria-selected:bg-text/5"
            tabindex="-1"
            (mousedown)="$event.preventDefault()"
            (click)="go(option.value)"
            (keydown.enter)="go(option.value)"
            (mouseenter)="active.set(index)"
          >
            <span class="icon-[heroicons--map-pin] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
            {{ option.label }}
            <span class="ml-auto text-xs text-text-muted">{{ 'nav.tours' | transloco }}</span>
          </li>
        }
      </ul>
    }
  `,
})
export class DestinationSearch {
  readonly variant = input<'hero' | 'field'>('hero');
  // A destination was chosen (the header closes its panel).
  readonly chosen = output();

  private readonly transloco = inject(TranslocoService);
  // Labels are matched in the visitor's language, once it has loaded.
  private readonly dictionary = toSignal(this.transloco.selectTranslation());
  private readonly navigate = injectSiteNavigate();
  private readonly field = viewChild<ElementRef<HTMLInputElement>>('field');
  protected readonly listId = `destination-list-${nextId++}`;
  protected readonly query = signal('');
  protected readonly open = signal(false);
  protected readonly active = signal(-1);

  protected readonly matches = computed(() => {
    const query = this.query().trim().toLocaleLowerCase();
    this.dictionary();
    return DESTINATION_OPTIONS.map((option) => ({
      value: option.value,
      label: this.transloco.translate(option.labelKey),
    })).filter((option) => !query || option.label.toLocaleLowerCase().includes(query));
  });

  focus(): void {
    this.field()?.nativeElement.focus();
  }

  protected onInput(value: string): void {
    this.query.set(value);
    this.open.set(true);
    this.active.set(value.trim() ? 0 : -1);
  }

  protected close(): void {
    this.open.set(false);
    this.active.set(-1);
  }

  protected onKeydown(event: KeyboardEvent): void {
    const count = this.matches().length;
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      this.open.set(true);
      if (count === 0) return;
      const step = event.key === 'ArrowDown' ? 1 : -1;
      this.active.set((this.active() + step + count) % count);
    } else if (event.key === 'Escape') {
      this.close();
    }
  }

  protected submit(event: Event): void {
    event.preventDefault();
    const options = this.matches();
    const pick = options[this.active()] ?? (this.query().trim() ? options[0] : undefined);
    if (pick) this.go(pick.value);
    else void this.navigate('/tours').then(() => this.chosen.emit());
  }

  protected go(destination: string): void {
    this.close();
    this.query.set('');
    void this.navigate(withQuery('/tours', { destination })).then(() => this.chosen.emit());
  }
}
