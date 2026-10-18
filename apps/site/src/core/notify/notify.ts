import { isPlatformBrowser } from '@angular/common';
import { Injectable, PLATFORM_ID, inject, signal } from '@angular/core';

export type ToastKind = 'success' | 'info' | 'warning' | 'error';

// What a toast says: translation keys (Transloco) and their parameters.
export interface ToastMessage {
  title: string;
  text?: string;
  params?: Record<string, string | number>;
  // A link-like button inside the toast ("Undo", "Retry").
  action?: { label: string; run: () => void };
  // Overrides the kind's default lifetime; null keeps it until closed.
  duration?: number | null;
}

export interface Toast extends ToastMessage {
  id: number;
  kind: ToastKind;
  duration: number | null;
  // The same message shown again: "×2" instead of a second toast.
  count: number;
}

// How long each kind stays, in ms; errors stay until closed.
const DURATIONS: Record<ToastKind, number | null> = {
  success: 4000,
  info: 5000,
  warning: 7000,
  error: null,
};

// At most this many at once; the oldest goes first.
const MAX_TOASTS = 3;

// Short messages about what just happened (saved, sent, failed), shown by
// ToastOutlet in the page's corner. Browser only: on the server nothing is
// queued.
@Injectable({ providedIn: 'root' })
export class Notify {
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID));
  private readonly list = signal<Toast[]>([]);
  readonly toasts = this.list.asReadonly();
  private nextId = 1;

  success(message: ToastMessage): number {
    return this.show('success', message);
  }

  info(message: ToastMessage): number {
    return this.show('info', message);
  }

  warning(message: ToastMessage): number {
    return this.show('warning', message);
  }

  error(message: ToastMessage): number {
    return this.show('error', message);
  }

  show(kind: ToastKind, message: ToastMessage): number {
    if (!this.browser) return 0;
    const same = this.list().find(
      (toast) =>
        toast.kind === kind && toast.title === message.title && toast.text === message.text,
    );
    if (same) {
      // Shown again: count it, and start its time over (a new id restarts the bar).
      const id = this.nextId++;
      this.list.update((list) =>
        list.map((toast) => (toast === same ? { ...toast, id, count: toast.count + 1 } : toast)),
      );
      return id;
    }
    const toast: Toast = {
      ...message,
      id: this.nextId++,
      kind,
      duration: message.duration === undefined ? DURATIONS[kind] : message.duration,
      count: 1,
    };
    this.list.update((list) => [toast, ...list].slice(0, MAX_TOASTS));
    return toast.id;
  }

  dismiss(id: number): void {
    this.list.update((list) => list.filter((toast) => toast.id !== id));
  }
}
