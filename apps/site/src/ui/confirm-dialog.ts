import { DIALOG_DATA, Dialog, DialogRef } from '@angular/cdk/dialog';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

export interface ConfirmOptions {
  // Translation keys and their parameters.
  title: string;
  text?: string;
  params?: Record<string, string | number>;
  confirmLabel?: string;
  // A destructive action: red button and a trash icon.
  danger?: boolean;
  icon?: string;
}

// "Delete this tour?" — instead of the browser's confirm(): the page's look,
// translated buttons, and focus on the safe choice (Cancel).
@Component({
  selector: 'app-confirm-dialog',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div
      class="motion-dialog-in w-[min(28rem,calc(100vw-2rem))] rounded-card border border-border bg-surface p-5 text-text shadow-[0_24px_50px_-12px_rgb(15_23_42/0.35)]"
      role="alertdialog"
      aria-labelledby="confirm-title"
      [attr.aria-describedby]="data.text ? 'confirm-text' : null"
    >
      <div class="flex items-start gap-3">
        <span
          class="flex size-10 shrink-0 items-center justify-center rounded-full"
          [class]="data.danger ? 'bg-red-500/12 text-danger' : 'bg-brand-500/10 text-brand-600 dark:text-brand-300'"
        >
          <span [class]="(data.icon ?? (data.danger ? 'icon-[heroicons--trash]' : 'icon-[heroicons--question-mark-circle]')) + ' size-5'" aria-hidden="true"></span>
        </span>
        <div>
          <h2 id="confirm-title" class="text-base font-semibold">{{ data.title | transloco: data.params }}</h2>
          @if (data.text) {
            <p id="confirm-text" class="mt-1 text-sm text-text-muted">{{ data.text | transloco: data.params }}</p>
          }
        </div>
      </div>
      <div class="mt-4 flex justify-end gap-2">
        <button type="button" class="ui-btn-soft" cdkFocusInitial (click)="ref.close(false)">{{ 'notify.cancel' | transloco }}</button>
        <button
          type="button"
          class="ui-btn-primary"
          [class.!bg-danger]="data.danger"
          (click)="ref.close(true)"
        >
          {{ (data.confirmLabel ?? (data.danger ? 'notify.delete' : 'notify.confirm')) | transloco }}
        </button>
      </div>
    </div>
  `,
})
export class ConfirmDialog {
  protected readonly data = inject<ConfirmOptions>(DIALOG_DATA);
  protected readonly ref = inject<DialogRef<boolean>>(DialogRef);
}

// Asks, and resolves to whether the visitor confirmed.
export function injectConfirm(): (options: ConfirmOptions) => Promise<boolean> {
  const dialog = inject(Dialog);
  return async (options) => {
    const ref = dialog.open<boolean>(ConfirmDialog, {
      data: options,
      backdropClass: ['bg-black/50', 'motion-backdrop-in'],
      ariaModal: true,
    });
    return (await firstValueFrom(ref.closed)) === true;
  };
}
