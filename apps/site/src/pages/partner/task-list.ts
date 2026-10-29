import type { CrmTask } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { clientName } from '@/pages/partner/crm';
import { FieldErrors } from '@/ui/field-errors';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

// The partner's tasks ("call back", "remind about the balance"): open ones
// by due day — overdue in red — with a box to add one. For one client when
// clientId is set; `limit` shows only the first ones (the dashboard).
@Component({
  selector: 'app-task-list',
  imports: [FieldErrors, FormField, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'flex flex-col gap-2' },
  template: `
    @if (shown().length > 0) {
      <ul class="flex flex-col divide-y divide-border">
        @for (task of shown(); track task.id) {
          <li class="flex items-start gap-2.5 py-2">
            <input type="checkbox" class="mt-1" [checked]="task.done" [attr.aria-label]="'crm.tasks.done' | transloco" [disabled]="busy()" (change)="toggle(task)" />
            <div class="min-w-0 flex-1 text-sm">
              <p class="text-text" [class.line-through]="task.done">{{ task.title }}</p>
              <p class="flex flex-wrap gap-x-2 text-xs" [class]="!task.done && task.dueOn < today ? 'text-danger' : 'text-text-muted'">
                <span>{{ due(task.dueOn) }}</span>
                @if (task.client && !clientId()) {
                  <a [appSiteLink]="'/partner/clients/' + task.client.id" class="text-text-muted hover:underline">{{ name(task) }}</a>
                }
                @if (task.tour) {
                  <a [appSiteLink]="'/partner/tours/' + task.tour.id + '/participants'" class="text-text-muted hover:underline">{{ task.tour.title }}</a>
                }
              </p>
            </div>
            <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.tasks.delete' | transloco" [disabled]="busy()" (click)="remove(task)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
          </li>
        }
      </ul>
    } @else if (tasks.hasValue()) {
      <p class="text-sm text-text-muted">{{ 'crm.tasks.none' | transloco }}</p>
    }
    @if (adding()) {
      <form class="flex flex-wrap items-start gap-2" novalidate (submit)="add($event)">
        <div class="min-w-48 flex-1">
          <label class="sr-only" for="task-title">{{ 'crm.tasks.title' | transloco }}</label>
          <input id="task-title" class="ui-input" [placeholder]="'crm.tasks.placeholder' | transloco" [formField]="taskForm.title" />
          <app-field-errors [state]="taskForm.title()" />
        </div>
        <label class="sr-only" for="task-due">{{ 'crm.tasks.due' | transloco }}</label>
        <input id="task-due" class="ui-input w-40" type="date" [formField]="taskForm.dueOn" />
        <button type="submit" class="ui-btn-primary" [disabled]="taskForm().submitting()">{{ 'crm.tasks.add' | transloco }}</button>
      </form>
    }
  `,
})
export class TaskList {
  readonly clientId = input<string | null>(null);
  readonly tourId = input<string | null>(null);
  readonly limit = input<number | null>(null);
  readonly adding = input(true);

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly today = todayInBishkek();
  protected readonly busy = signal(false);

  protected readonly tasks = httpResource<CrmTask[]>(() => {
    const clientId = this.clientId();
    return { url: '/crm/tasks', params: clientId ? { clientId } : undefined };
  });
  protected readonly shown = computed(() => {
    const all = loaded(this.tasks) ?? [];
    const limit = this.limit();
    return limit === null ? all : all.slice(0, limit);
  });

  private readonly model = linkedSignal(() => ({ title: '', dueOn: this.today }));
  protected readonly taskForm = form(this.model, (path) => {
    validateStandardSchema(
      path,
      z.object({
        title: z
          .string()
          .check(
            z.trim(),
            z.minLength(1, this.transloco.translate('crm.tasks.empty')),
            z.maxLength(200, this.transloco.translate('crm.errors.tooLong')),
          ),
        dueOn: z.string().check(z.minLength(10, this.transloco.translate('crm.errors.date'))),
      }),
    );
  });

  protected name(task: CrmTask): string {
    return task.client ? clientName(task.client) : '';
  }

  protected due(day: string): string {
    if (day === this.today) return this.transloco.translate('crm.tasks.today');
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short' });
  }

  protected async add(event: Event): Promise<void> {
    event.preventDefault();
    await submit(this.taskForm, async () => {
      try {
        const v = this.model();
        await firstValueFrom(
          this.http.post<CrmTask[]>('/crm/tasks', {
            title: v.title.trim(),
            dueOn: v.dueOn,
            ...(this.clientId() ? { clientId: this.clientId() } : {}),
            ...(this.tourId() ? { tourId: this.tourId() } : {}),
          }),
        );
        this.model.set({ title: '', dueOn: this.today });
        this.taskForm().reset();
        this.tasks.reload();
      } catch {
        this.notify.error({ title: 'crm.saveError' });
      }
      return undefined;
    });
  }

  protected async toggle(task: CrmTask): Promise<void> {
    await this.change(this.http.patch<CrmTask[]>(`/crm/tasks/${task.id}`, { done: !task.done }));
  }

  protected async remove(task: CrmTask): Promise<void> {
    await this.change(this.http.delete<CrmTask[]>(`/crm/tasks/${task.id}`));
  }

  private async change(request: ReturnType<HttpClient['delete']>): Promise<void> {
    this.busy.set(true);
    try {
      await firstValueFrom(request);
      this.tasks.reload();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }
}
