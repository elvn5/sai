import type { CrmTemplate } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { FieldErrors } from '@/ui/field-errors';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

// A message to many clients at once: a tour's group, clients with a tag
// or all of them. Each with an account gets it in their chat with the
// partner; the text can start from a saved template.
@Component({
  selector: 'app-broadcast-form',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-3" novalidate (submit)="send($event)">
      @if (!audience()) {
        <div class="grid gap-2 sm:grid-cols-2">
          <div class="ui-field">
            <label class="ui-label" for="broadcast-audience">{{ 'crm.broadcast.audience' | transloco }}</label>
            <select id="broadcast-audience" class="ui-input" [value]="target()" (change)="target.set($any($event.target).value)">
              <option value="all">{{ 'crm.broadcast.all' | transloco }}</option>
              @for (t of tags(); track t) {
                <option [value]="'tag:' + t">{{ 'crm.broadcast.tag' | transloco: { tag: t } }}</option>
              }
              @for (tour of tours(); track tour.id) {
                <option [value]="'tour:' + tour.id">{{ 'crm.broadcast.tour' | transloco: { title: tour.title } }}</option>
              }
            </select>
          </div>
        </div>
      }
      @if (templateList().length > 0) {
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-xs text-text-muted">{{ 'crm.templates.insert' | transloco }}</span>
          @for (t of templateList(); track t.id) {
            <button type="button" class="ui-pill px-2.5 py-1 text-xs" (click)="useTemplate(t)">{{ t.title }}</button>
          }
        </div>
      }
      <div class="ui-field">
        <label class="ui-label" for="broadcast-body">{{ 'crm.broadcast.text' | transloco }}</label>
        <textarea id="broadcast-body" class="ui-input" rows="4" [formField]="messageForm.body"></textarea>
        <app-field-errors [state]="messageForm.body()" />
      </div>
      <p class="ui-hint">{{ 'crm.broadcast.hint' | transloco }}</p>
      <div class="flex gap-2">
        <button type="submit" class="ui-btn-primary" [disabled]="messageForm().submitting()"><span class="icon-[heroicons--paper-airplane] size-4" aria-hidden="true"></span>{{ 'crm.broadcast.send' | transloco }}</button>
        @if (cancellable()) {
          <button type="button" class="ui-btn-ghost" (click)="cancelled.emit()">{{ 'crm.cancel' | transloco }}</button>
        }
      </div>
    </form>
  `,
})
export class BroadcastForm {
  // A fixed audience ("tour:<id>"); none: the partner picks one.
  readonly audience = input<string | null>(null);
  readonly tags = input<string[]>([]);
  readonly tours = input<{ id: string; title: string }[]>([]);
  readonly cancellable = input(false);
  readonly sent = output<{ sent: number; skipped: number }>();
  readonly cancelled = output();

  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  protected readonly target = signal('all');
  private readonly templates = httpResource<CrmTemplate[]>(() => '/crm/templates');
  protected readonly templateList = computed(() => loaded(this.templates) ?? []);

  private readonly model = signal({ body: '' });
  protected readonly messageForm = form(this.model, (path) => {
    validateStandardSchema(
      path,
      z.object({
        body: z
          .string()
          .check(
            z.trim(),
            z.minLength(1, this.transloco.translate('crm.broadcast.empty')),
            z.maxLength(4000, this.transloco.translate('chat.tooLong')),
          ),
      }),
    );
  });

  protected useTemplate(t: CrmTemplate): void {
    this.model.update((m) => ({ body: m.body.trim() ? `${m.body.trim()}\n\n${t.body}` : t.body }));
  }

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    await submit(this.messageForm, async () => {
      try {
        const result = await firstValueFrom(
          this.http.post<{ sent: number; skipped: number }>('/crm/broadcasts', {
            audience: this.audience() ?? this.target(),
            body: this.model().body.trim(),
          }),
        );
        this.model.set({ body: '' });
        this.messageForm().reset();
        this.notify.success({ title: 'crm.broadcast.done', params: result });
        this.sent.emit(result);
      } catch {
        this.notify.error({ title: 'crm.saveError' });
      }
      return undefined;
    });
  }
}
