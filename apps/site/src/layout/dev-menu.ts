import { APP_CONFIG } from '@/core/config/app-config';
import { checkDevPassword } from '@/core/feature-flags/dev-menu-password';
import { FEATURE_FLAGS, type FeatureFlag, FeatureFlags } from '@/core/feature-flags/feature-flags';
import { Notify } from '@/core/notify/notify';
import { Dialog, DialogRef } from '@angular/cdk/dialog';
import { ChangeDetectionStrategy, Component, type Injector, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Unlocked once per page load: the menu opens again without the password.
const unlocked = signal(false);

// The developer menu, behind a password: switches that turn parts of the
// site off in this browser (core/feature-flags). Opened by a triple click on
// the version in the footer.
@Component({
  selector: 'app-dev-menu',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div
      class="motion-dialog-in flex max-h-[min(40rem,calc(100dvh-2rem))] w-[min(30rem,calc(100vw-2rem))] flex-col rounded-card border border-border bg-surface text-text shadow-[0_24px_50px_-12px_rgb(15_23_42/0.35)]"
      role="dialog"
      aria-labelledby="dev-menu-title"
    >
      <header class="flex items-center gap-3 p-5 pb-3">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-brand-500/10 text-brand-600 dark:text-brand-300">
          <span [class]="(open() ? 'icon-[heroicons--wrench-screwdriver]' : 'icon-[heroicons--lock-closed]') + ' size-5'" aria-hidden="true"></span>
        </span>
        <h2 id="dev-menu-title" class="text-base font-semibold">{{ (open() ? 'devMenu.title' : 'devMenu.password.title') | transloco }}</h2>
        <button type="button" class="ui-icon-btn -mr-1.5 ml-auto size-8" [attr.aria-label]="'notify.close' | transloco" (click)="ref.close()">
          <span class="icon-[heroicons--x-mark] size-5" aria-hidden="true"></span>
        </button>
      </header>

      @if (!open()) {
        <form class="px-5 pb-5" (submit)="unlock($event, passwordInput.value)">
          <label class="block">
            <span class="ui-label">{{ 'devMenu.password.label' | transloco }}</span>
            <input
              #passwordInput
              class="ui-input"
              type="password"
              autocomplete="off"
              cdkFocusInitial
              [attr.aria-invalid]="wrong()"
              [attr.aria-describedby]="wrong() ? 'dev-menu-error' : null"
              (input)="wrong.set(false)"
            />
          </label>
          @if (wrong()) {
            <p id="dev-menu-error" class="mt-1.5 text-sm text-danger" role="alert">{{ 'devMenu.password.wrong' | transloco }}</p>
          }
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" class="ui-btn-soft" (click)="ref.close()">{{ 'notify.cancel' | transloco }}</button>
            <button type="submit" class="ui-btn-primary" [disabled]="checking()" [attr.aria-busy]="checking()">{{ 'devMenu.password.submit' | transloco }}</button>
          </div>
        </form>
      } @else {
        <p class="px-5 text-sm text-text-muted">{{ 'devMenu.hint' | transloco }}</p>
        <ul class="mt-3 min-h-0 flex-1 divide-y divide-border overflow-y-auto border-y border-border">
          @for (flag of flagList; track flag) {
            <li>
              <label class="flex cursor-pointer items-start gap-3 px-5 py-3 hover:bg-text/4">
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-medium">{{ 'devMenu.flags.' + flag + '.title' | transloco }}</span>
                  <span class="block text-xs text-text-muted">{{ 'devMenu.flags.' + flag + '.text' | transloco }}</span>
                </span>
                <input
                  type="checkbox"
                  role="switch"
                  class="peer sr-only"
                  [checked]="flags.isOn(flag)"
                  (change)="toggle(flag, $event)"
                />
                <span
                  class="relative mt-0.5 h-6 w-10 shrink-0 rounded-full bg-border transition-colors peer-checked:bg-brand-600 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand-500 after:absolute after:top-0.5 after:left-0.5 after:size-5 after:rounded-full after:bg-white after:shadow-sm after:transition-transform peer-checked:after:translate-x-4 motion-reduce:transition-none motion-reduce:after:transition-none"
                  aria-hidden="true"
                ></span>
              </label>
            </li>
          }
        </ul>
        <footer class="flex items-center justify-between gap-2 p-5 pt-4">
          <button type="button" class="ui-btn-ghost" [disabled]="flags.changedCount() === 0" (click)="reset()">
            <span class="icon-[heroicons--arrow-path] size-4" aria-hidden="true"></span>{{ 'devMenu.reset' | transloco }}
          </button>
          <button type="button" class="ui-btn-primary" (click)="ref.close()">{{ 'devMenu.done' | transloco }}</button>
        </footer>
      }
    </div>
  `,
})
export class DevMenu {
  protected readonly ref = inject<DialogRef<void>>(DialogRef);
  protected readonly flags = inject(FeatureFlags);
  private readonly notify = inject(Notify);
  private readonly hash = inject(APP_CONFIG).devMenuPasswordHash;
  protected readonly flagList = FEATURE_FLAGS;
  protected readonly open = unlocked.asReadonly();
  protected readonly checking = signal(false);
  protected readonly wrong = signal(false);

  protected async unlock(event: Event, password: string): Promise<void> {
    event.preventDefault();
    this.checking.set(true);
    const ok = await checkDevPassword(password, this.hash);
    this.checking.set(false);
    this.wrong.set(!ok);
    if (ok) unlocked.set(true);
  }

  protected toggle(flag: FeatureFlag, event: Event): void {
    this.flags.set(flag, (event.target as HTMLInputElement).checked);
  }

  protected reset(): void {
    this.flags.reset();
    this.notify.success({ title: 'devMenu.resetDone' });
  }
}

// Opens the menu (the footer loads this file on the triple click).
export function openDevMenu(injector: Injector): void {
  injector.get(Dialog).open(DevMenu, {
    backdropClass: ['bg-black/50', 'motion-backdrop-in'],
    ariaModal: true,
    ariaLabelledBy: 'dev-menu-title',
  });
}
