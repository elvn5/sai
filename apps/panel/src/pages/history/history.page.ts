import { HISTORY_MODULES, type HistoryModule } from '@/core/history/history';
import { HistoryLog } from '@/ui/history-log';
import { ChangeDetectionStrategy, Component, computed, signal } from '@angular/core';

// Every module's changes in one list, or one module's.
@Component({
  selector: 'app-history-page',
  imports: [HistoryLog],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mb-4 flex items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold text-text">История</h1>
        <p class="text-sm text-text-muted">Изменения, сделанные в админке, — новые сверху.</p>
      </div>
      <select class="ui-input w-48" aria-label="Раздел" [value]="module()" (change)="module.set($any($event.target).value)">
        <option value="">Все разделы</option>
        @for (option of options; track option.module) {
          <option [value]="option.module">{{ option.label }}</option>
        }
      </select>
    </div>
    <app-history-log [modules]="modules()" [open]="true" title="Журнал изменений" />
  `,
})
export class HistoryPage {
  protected readonly options = HISTORY_MODULES;
  protected readonly module = signal<HistoryModule | ''>('');
  protected readonly modules = computed<HistoryModule[]>(() => {
    const module = this.module();
    return module ? [module] : HISTORY_MODULES.map((m) => m.module);
  });
}
