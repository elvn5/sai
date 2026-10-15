import { DEFAULT_LANG, type Lang, isLang } from '@/core/i18n/langs';
import { type Signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { TranslocoService } from '@jsverse/transloco';
import type { Observable } from 'rxjs';

// The active language as a signal.
export function injectLang(): Signal<Lang> {
  const transloco = inject(TranslocoService);
  const active = transloco.getActiveLang();
  return toSignal(transloco.langChanges$ as Observable<Lang>, {
    initialValue: isLang(active) ? active : DEFAULT_LANG,
  });
}
