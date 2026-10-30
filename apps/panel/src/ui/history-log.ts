import {
  HISTORY_MODULES,
  type HistoryEntry,
  type HistoryModule,
  actionLabel,
  actionTone,
  mergeHistory,
  moduleLabel,
  subjectLabel,
} from '@/core/history/history';
import { formatDateTime } from '@/core/lib/format';
import { JsonPipe } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

const PAGE = 30;

// The change history of one or more modules, newest first, paged with
// "Показать ещё". Collapsed, it loads nothing until opened.
@Component({
  selector: 'app-history-log',
  imports: [JsonPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <details class="ui-card" [open]="open()" (toggle)="onToggle($event)">
      <summary class="flex cursor-pointer items-center gap-2 px-4 py-3 text-sm font-medium text-text">
        <span class="icon-[heroicons--clock] size-4 text-text-muted" aria-hidden="true"></span>
        {{ title() }}
      </summary>
      <div class="overflow-x-auto border-t border-border">
        <table class="ui-table">
          <thead>
            <tr>
              <th>Когда</th>
              @if (showModule()) {
                <th>Раздел</th>
              }
              <th>Действие</th>
              <th>Объект</th>
              <th>Откуда</th>
              <th><span class="sr-only">Подробности</span></th>
            </tr>
          </thead>
          <tbody>
            @for (entry of entries(); track entry.id) {
              <tr>
                <td class="whitespace-nowrap text-text-muted">{{ formatDateTime(entry.createdAt) }}</td>
                @if (showModule()) {
                  <td>{{ moduleLabel(entry.module) }}</td>
                }
                <td><span class="ui-badge whitespace-nowrap" [class]="actionTone(entry)">{{ actionLabel(entry) }}</span></td>
                <td class="max-w-56 truncate" [title]="entry.entityId ?? ''">{{ subject(entry) }}</td>
                <td class="whitespace-nowrap text-xs text-text-muted" [title]="entry.userAgent">{{ entry.ip || '—' }}</td>
                <td class="text-right">
                  @if (entry.payload !== null) {
                    <button
                      type="button"
                      class="ui-btn-ghost ui-btn-xs"
                      [attr.aria-expanded]="expanded() === entry.id"
                      (click)="toggle(entry.id)"
                    >
                      Данные
                    </button>
                  }
                </td>
              </tr>
              @if (expanded() === entry.id) {
                <tr>
                  <td [attr.colspan]="columns()" class="bg-text/5">
                    <p class="mb-1 text-xs text-text-muted">{{ entry.method }} {{ entry.path }}</p>
                    <pre class="max-h-80 overflow-auto whitespace-pre-wrap break-all text-xs">{{ entry.payload | json }}</pre>
                  </td>
                </tr>
              }
            } @empty {
              @if (!loading()) {
                <tr><td [attr.colspan]="columns()" class="ui-empty">Изменений пока нет</td></tr>
              }
            }
          </tbody>
        </table>
        @if (error()) {
          <p class="ui-error px-4 py-3">{{ error() }}</p>
        }
        @if (hasMore() || loading()) {
          <div class="border-t border-border px-4 py-3 text-center">
            <button type="button" class="ui-btn-soft" [disabled]="loading()" (click)="loadMore()">
              {{ loading() ? 'Загрузка…' : 'Показать ещё' }}
            </button>
          </div>
        }
      </div>
    </details>
  `,
})
export class HistoryLog {
  private readonly http = inject(HttpClient);

  readonly modules = input.required<HistoryModule[]>();
  readonly title = input('История изменений');
  readonly open = input(false);
  // Only the changes of this entity (e.g. one partner).
  readonly entityId = input<string | null>(null);
  // Names the page knows for entity ids (e.g. people for partner ids),
  // used when the change itself names nothing.
  readonly names = input<Record<string, string>>({});

  protected readonly entries = signal<HistoryEntry[]>([]);
  protected readonly loading = signal(false);
  protected readonly hasMore = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly expanded = signal<string | null>(null);
  private readonly opened = signal(false);
  private readonly active = computed(() => this.open() || this.opened());
  // Bumped on every restart: an answer for an older one is dropped.
  private generation = 0;

  protected readonly showModule = computed(() => this.modules().length > 1);
  protected readonly columns = computed(() => (this.showModule() ? 6 : 5));

  protected readonly formatDateTime = formatDateTime;
  protected readonly moduleLabel = moduleLabel;
  protected readonly actionLabel = actionLabel;
  protected readonly actionTone = actionTone;

  protected subject(entry: HistoryEntry): string {
    const label = subjectLabel(entry);
    return (label === entry.entityId && this.names()[label]) || label;
  }

  constructor() {
    // Starts over when the modules or the entity change, once opened.
    effect(() => {
      this.modules();
      this.entityId();
      if (this.active()) untracked(() => this.reload());
    });
  }

  // Reloads from the newest entry, e.g. after the page made a change.
  reload(): void {
    if (!this.active()) return;
    this.generation++;
    this.entries.set([]);
    this.hasMore.set(false);
    void this.loadMore();
  }

  protected onToggle(event: Event): void {
    if ((event.target as HTMLDetailsElement).open) this.opened.set(true);
  }

  protected toggle(id: string): void {
    this.expanded.update((current) => (current === id ? null : id));
  }

  protected async loadMore(): Promise<void> {
    const generation = this.generation;
    this.loading.set(true);
    this.error.set(null);
    const before = this.entries().at(-1)?.createdAt;
    const entityId = this.entityId();
    const params = {
      limit: String(PAGE),
      ...(before ? { before } : {}),
      ...(entityId ? { entityId } : {}),
    };
    const sources = HISTORY_MODULES.filter((m) => this.modules().includes(m.module));
    // Each module is its own service: one that's down mustn't hide the rest.
    const results = await Promise.allSettled(
      sources.map((m) => firstValueFrom(this.http.get<HistoryEntry[]>(m.url, { params }))),
    );
    if (generation !== this.generation) return;
    const pages = results.flatMap((r) => (r.status === 'fulfilled' ? [r.value] : []));
    const failed = sources.filter((_, i) => results[i]?.status === 'rejected').map((m) => m.label);
    if (failed.length) {
      this.error.set(
        sources.length === 1
          ? 'Не удалось загрузить историю'
          : `Не удалось загрузить историю: ${failed.join(', ')}`,
      );
    }
    this.entries.update((entries) => [...entries, ...mergeHistory(pages, PAGE)]);
    this.hasMore.set(pages.some((p) => p.length === PAGE) || pages.flat().length > PAGE);
    this.loading.set(false);
  }
}
