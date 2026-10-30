import type { ConditionReportSummary, TrailSummary } from '@/core/api/types';
import { HistoryLog } from '@/ui/history-log';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-trails-page',
  imports: [HistoryLog],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-8">
      <section>
        <h1 class="text-xl font-semibold text-text">Маршруты</h1>
        <p class="mb-4 text-sm text-text-muted">Маршруты публикуют партнёры на сайте. Здесь можно скрыть или удалить любой.</p>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th>Название</th>
                <th>Район</th>
                <th>Категория</th>
                <th>Статус</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (trail of trails.value() ?? []; track trail.id) {
                <tr>
                  <td>
                    {{ trail.name }}
                    <span class="ml-1 text-xs text-text-muted">/trails/{{ trail.slug }}</span>
                    @if (!trail.userId) {
                      <span class="ui-badge ml-2 bg-text/5 text-text-muted">редакция</span>
                    }
                  </td>
                  <td class="text-text-muted">{{ trail.area }}</td>
                  <td class="text-text-muted">{{ trail.grade }}</td>
                  <td class="text-text-muted">{{ trail.isPublished ? 'Опубликован' : 'Скрыт' }}</td>
                  <td class="whitespace-nowrap text-right">
                    <button type="button" class="ui-btn-ghost ui-btn-xs" [disabled]="busyId() === trail.id" (click)="togglePublished(trail)">
                      {{ trail.isPublished ? 'Скрыть' : 'Опубликовать' }}
                    </button>
                    <button type="button" class="ui-btn-danger ui-btn-xs" aria-label="Удалить маршрут" (click)="removeTrail(trail)">
                      <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
                    </button>
                  </td>
                </tr>
              } @empty {
                @if (!trails.isLoading()) {
                  <tr><td colspan="5" class="ui-empty">Маршрутов пока нет</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-semibold text-text">Последние сводки</h2>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th>Маршрут</th>
                <th>Автор</th>
                <th>Дата</th>
                <th>Текст</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (report of reports.value() ?? []; track report.id) {
                <tr>
                  <td>{{ report.trail.name }}</td>
                  <td class="text-text-muted">{{ report.author.firstName ?? report.author.username ?? '—' }}</td>
                  <td class="text-text-muted">{{ report.observedOn }}</td>
                  <td class="max-w-md truncate text-text-muted">{{ report.text ?? report.trailState + ' · ' + report.traction }}</td>
                  <td class="text-right">
                    <button type="button" class="ui-btn-danger ui-btn-xs" aria-label="Удалить сводку" (click)="removeReport(report)">
                      <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
                    </button>
                  </td>
                </tr>
              } @empty {
                @if (!reports.isLoading()) {
                  <tr><td colspan="5" class="ui-empty">Сводок пока нет</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <app-history-log class="mt-8" [modules]="['trails']" />
  `,
})
export class TrailsPage {
  private readonly http = inject(HttpClient);
  protected readonly trails = httpResource<TrailSummary[]>(() => '/trails/admin/all');
  protected readonly reports = httpResource<ConditionReportSummary[]>(() => '/conditions?limit=50');
  protected readonly busyId = signal<string | null>(null);

  private reload(): void {
    this.trails.reload();
    this.reports.reload();
  }

  protected async togglePublished(trail: TrailSummary): Promise<void> {
    this.busyId.set(trail.id);
    try {
      await firstValueFrom(
        this.http.put(`/trails/${trail.id}`, { isPublished: !trail.isPublished }),
      );
      this.reload();
    } finally {
      this.busyId.set(null);
    }
  }

  protected async removeTrail(trail: TrailSummary): Promise<void> {
    if (!window.confirm(`Удалить маршрут «${trail.name}» вместе со сводками?`)) return;
    await firstValueFrom(this.http.delete(`/trails/${trail.id}`));
    this.reload();
  }

  protected async removeReport(report: ConditionReportSummary): Promise<void> {
    if (!window.confirm('Удалить сводку?')) return;
    await firstValueFrom(this.http.delete(`/conditions/${report.id}`));
    this.reload();
  }
}
