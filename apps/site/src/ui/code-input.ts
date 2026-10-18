import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  computed,
  input,
  model,
  viewChildren,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';

// A one-time code as six boxes: typing moves on, Backspace moves back, a
// pasted code fills them all (and the phone's SMS/email suggestion does too:
// autocomplete="one-time-code" on the first box).
@Component({
  selector: 'app-code-input',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="grid grid-cols-6 gap-1.5" role="group" [attr.aria-label]="label()" [class.motion-shake]="refused()">
      @for (index of slots; track index) {
        <input
          #box
          class="ui-input h-12 px-0 text-center text-xl font-bold tabular-nums"
          [class.!border-danger]="refused()"
          inputmode="numeric"
          [attr.maxlength]="index === 0 ? 6 : 1"
          [attr.autocomplete]="index === 0 ? 'one-time-code' : 'off'"
          [attr.aria-label]="label() + ' ' + (index + 1)"
          [value]="(digits()[index] ?? '').trim()"
          (input)="onInput(index, $any($event.target))"
          (keydown)="onKeydown(index, $event)"
          (paste)="onPaste(index, $event)"
          (focus)="$any($event.target).select()"
        />
      }
    </div>
  `,
})
export class CodeInput implements FormValueControl<string> {
  readonly value = model('');
  readonly refused = input(false);
  readonly label = input('');
  protected readonly slots = [0, 1, 2, 3, 4, 5];
  private readonly boxes = viewChildren<ElementRef<HTMLInputElement>>('box');
  protected readonly digits = computed(() => this.value().split(''));

  focus(): void {
    this.boxes()[0]?.nativeElement.focus();
  }

  protected onInput(index: number, box: HTMLInputElement): void {
    const digits = box.value.replace(/\D/g, '');
    if (digits.length > 1) {
      // The phone filled in the whole code.
      this.fill(index, digits);
      return;
    }
    this.set(index, digits);
    if (digits) this.boxes()[index + 1]?.nativeElement.focus();
  }

  protected onKeydown(index: number, event: KeyboardEvent): void {
    if (event.key === 'Backspace' && !this.digits()[index] && index > 0) {
      event.preventDefault();
      this.set(index - 1, '');
      this.boxes()[index - 1]?.nativeElement.focus();
    } else if (event.key === 'ArrowLeft' && index > 0) {
      this.boxes()[index - 1]?.nativeElement.focus();
    } else if (event.key === 'ArrowRight') {
      this.boxes()[index + 1]?.nativeElement.focus();
    }
  }

  protected onPaste(index: number, event: ClipboardEvent): void {
    const digits = (event.clipboardData?.getData('text') ?? '').replace(/\D/g, '');
    if (!digits) return;
    event.preventDefault();
    this.fill(index, digits);
  }

  private fill(from: number, digits: string): void {
    const chars = this.value().padEnd(6, ' ').split('');
    for (let i = 0; i < digits.length && from + i < 6; i++) chars[from + i] = digits[i] ?? ' ';
    this.value.set(chars.join('').trimEnd());
    this.boxes()[Math.min(from + digits.length, 5)]?.nativeElement.focus();
  }

  private set(index: number, digit: string): void {
    const chars = this.value().padEnd(6, ' ').split('');
    chars[index] = digit || ' ';
    this.value.set(chars.join('').trimEnd());
  }
}
