import { Notify, type Toast, type ToastKind } from '@/core/notify/notify';
import { DOCUMENT } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  computed,
  inject,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const ICONS: Record<ToastKind, string> = {
  success: 'icon-[heroicons--check]',
  info: 'icon-[heroicons--information-circle]',
  warning: 'icon-[heroicons--exclamation-triangle]',
  error: 'icon-[heroicons--x-circle]',
};

const TONES: Record<ToastKind, string> = {
  success: 'bg-green-500/14 text-success',
  info: 'bg-sky-500/13 text-info',
  warning: 'bg-amber-500/16 text-warning',
  error: 'bg-red-500/12 text-danger',
};

const BARS: Record<ToastKind, string> = {
  success: 'bg-success',
  info: 'bg-info',
  warning: 'bg-warning',
  error: 'bg-danger',
};

// Where toasts appear: top right under the header on wide screens, at the
// bottom (above sticky bars) on phones. The lifetime is the bar's
// animation: hovering, focusing or leaving the tab pauses it, and when it
// ends the toast goes. Errors are announced at once (assertive), the rest
// politely.
@Component({
  selector: 'app-toast-outlet',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    @keyframes toast-time {
      from {
        transform: scaleX(1);
      }
      to {
        transform: scaleX(0);
      }
    }
    .bar {
      transform-origin: left;
      animation: toast-time linear both;
    }
    .toast:hover .bar,
    .toast:focus-within .bar,
    .paused .bar {
      animation-play-state: paused;
    }
    @media (width < 40rem) {
      .stack {
        --toast-from: 12px;
      }
    }
  `,
  template: `
    <div
      class="stack pointer-events-none fixed inset-x-3 bottom-[max(1rem,env(safe-area-inset-bottom))] z-[70] flex flex-col-reverse items-center gap-2 sm:inset-x-auto sm:top-20 sm:right-4 sm:bottom-auto sm:flex-col sm:items-end"
      [class.paused]="hidden()"
    >
      <div class="sr-only" role="status" aria-live="polite">
        @if (polite(); as toast) { {{ toast.title | transloco: toast.params }}. {{ toast.text ? (toast.text | transloco: toast.params) : '' }} }
      </div>
      <div class="sr-only" role="alert" aria-live="assertive">
        @if (assertive(); as toast) { {{ toast.title | transloco: toast.params }}. {{ toast.text ? (toast.text | transloco: toast.params) : '' }} }
      </div>
      @for (toast of notify.toasts(); track toast.id) {
        <div
          class="toast pointer-events-auto relative flex w-full max-w-sm items-start gap-2.5 overflow-hidden rounded-xl border border-border bg-surface py-3 pr-3 pl-3.5 text-sm text-text shadow-[0_12px_28px_-8px_rgb(15_23_42/0.25)]"
          animate.enter="motion-toast-in"
          animate.leave="motion-toast-out"
        >
          <span class="flex size-7 shrink-0 items-center justify-center rounded-full" [class]="tone(toast)">
            <span [class]="icon(toast) + ' size-[1.05rem]'" aria-hidden="true"></span>
          </span>
          <div class="min-w-0 flex-1">
            <p class="text-[0.84rem] font-semibold">
              {{ toast.title | transloco: toast.params }}
              @if (toast.count > 1) {
                <span class="ml-1 text-text-muted">×{{ toast.count }}</span>
              }
            </p>
            @if (toast.text) {
              <p class="text-[0.8rem] text-text-muted">{{ toast.text | transloco: toast.params }}</p>
            }
            @if (toast.action; as action) {
              <button
                type="button"
                class="mt-1 text-[0.8rem] font-semibold text-brand-700 hover:underline dark:text-brand-300"
                (click)="act(toast)"
              >
                {{ action.label | transloco }}
              </button>
            }
          </div>
          <button
            type="button"
            class="-m-1 flex size-7 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-text/5 hover:text-text"
            [attr.aria-label]="'notify.close' | transloco"
            (click)="notify.dismiss(toast.id)"
          >
            <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>
          </button>
          @if (toast.duration; as duration) {
            <span
              class="bar absolute inset-x-0 bottom-0 h-[3px]"
              [class]="bar(toast)"
              [style.animation-duration.ms]="duration"
              (animationend)="notify.dismiss(toast.id)"
            ></span>
          }
        </div>
      }
    </div>
  `,
})
export class ToastOutlet {
  protected readonly notify = inject(Notify);
  // The tab is in the background: time stands still.
  protected readonly hidden = signal(false);

  // The live regions read out the newest message.
  private readonly newest = computed(() => this.notify.toasts()[0]);
  protected readonly polite = computed(() => {
    const toast = this.newest();
    return toast && toast.kind !== 'error' ? toast : null;
  });
  protected readonly assertive = computed(() => {
    const toast = this.newest();
    return toast?.kind === 'error' ? toast : null;
  });

  constructor() {
    const document = inject(DOCUMENT);
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const onVisibility = (): void => this.hidden.set(document.hidden);
      document.addEventListener('visibilitychange', onVisibility);
      destroyRef.onDestroy(() => document.removeEventListener('visibilitychange', onVisibility));
    });
  }

  protected icon(toast: Toast): string {
    return ICONS[toast.kind];
  }

  protected tone(toast: Toast): string {
    return TONES[toast.kind];
  }

  protected bar(toast: Toast): string {
    return BARS[toast.kind];
  }

  protected act(toast: Toast): void {
    toast.action?.run();
    this.notify.dismiss(toast.id);
  }
}
