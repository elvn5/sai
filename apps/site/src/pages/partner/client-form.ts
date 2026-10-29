import type { CrmClient } from '@/core/api/types';
import { CLIENT_SOURCES } from '@/pages/partner/crm';
import { FieldErrors } from '@/ui/field-errors';
import { SelectField, injectOptions } from '@/ui/select-field';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const PHONE = /^\+?[0-9\s\-()]{5,32}$/;
const TELEGRAM = /^@?[A-Za-z0-9_]{5,32}$/;

// A client's details as the partner keeps them: name, phone, Telegram, a
// note and tags (comma-separated). Creates a client added by hand, or edits
// one (for an account the name is optional: the account's shows).
@Component({
  selector: 'app-client-form',
  imports: [FieldErrors, FormField, SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-3" novalidate (submit)="save($event)">
      <div class="grid gap-3 sm:grid-cols-3">
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'name'">{{ 'crm.client.name' | transloco }}</label>
          <input class="ui-input" [id]="prefix() + 'name'" autocomplete="off" [placeholder]="account() ? ('crm.client.namePlaceholderAccount' | transloco) : ''" [formField]="clientForm.name" />
          <app-field-errors [state]="clientForm.name()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'phone'">{{ 'crm.client.phone' | transloco }}</label>
          <input class="ui-input" type="tel" [id]="prefix() + 'phone'" autocomplete="off" placeholder="+996 555 123 456" [formField]="clientForm.phone" />
          <app-field-errors [state]="clientForm.phone()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'telegram'">{{ 'crm.client.telegram' | transloco }}</label>
          <input class="ui-input" [id]="prefix() + 'telegram'" autocomplete="off" placeholder="@username" [formField]="clientForm.telegram" />
          <app-field-errors [state]="clientForm.telegram()" />
        </div>
      </div>
      <div class="ui-field">
        <span class="ui-label" [id]="prefix() + 'source'">{{ 'crm.client.source' | transloco }}</span>
        <app-select-field [options]="sourceOptions()" [labelledBy]="prefix() + 'source'" [formField]="clientForm.source" />
      </div>
      <div class="ui-field">
        <label class="ui-label" [for]="prefix() + 'tags'">{{ 'crm.client.tags' | transloco }}</label>
        <input class="ui-input" [id]="prefix() + 'tags'" autocomplete="off" [placeholder]="'crm.client.tagsPlaceholder' | transloco" [formField]="clientForm.tags" />
      </div>
      <div class="ui-field">
        <label class="ui-label" [for]="prefix() + 'note'">{{ 'crm.client.note' | transloco }}</label>
        <textarea class="ui-input" rows="3" [id]="prefix() + 'note'" [placeholder]="'crm.client.notePlaceholder' | transloco" [formField]="clientForm.note"></textarea>
        <app-field-errors [state]="clientForm.note()" />
      </div>
      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'crm.saveError' | transloco }}</p>
      }
      <div class="flex gap-2">
        <button type="submit" class="ui-btn-primary" [disabled]="clientForm().submitting()">{{ (client() ? 'crm.save' : 'crm.client.add') | transloco }}</button>
        <button type="button" class="ui-btn-ghost" (click)="cancelled.emit()">{{ 'crm.cancel' | transloco }}</button>
      </div>
    </form>
  `,
})
export class ClientForm {
  // The client to edit; none: a new client added by hand.
  readonly client = input<CrmClient | null>(null);
  readonly saved = output<CrmClient>();
  readonly cancelled = output();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  protected readonly failed = signal(false);
  protected readonly sourceOptions = injectOptions(() => [
    { value: '', key: 'crm.source.unknown' },
    ...CLIENT_SOURCES.map((s) => ({ value: s, key: `crm.source.${s}` })),
  ]);
  protected readonly account = computed(() => this.client()?.userId != null);
  protected readonly prefix = computed(() => `client-${this.client()?.id ?? 'new'}-`);

  private readonly model = linkedSignal(() => {
    const c = this.client();
    return {
      name: c?.name ?? '',
      phone: c?.phone ?? '',
      telegram: c?.telegram ? `@${c.telegram}` : '',
      note: c?.note ?? '',
      tags: c?.tags.join(', ') ?? '',
      source: c?.source ?? '',
    };
  });

  protected readonly clientForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const optional = (pattern: RegExp, message: string) =>
      z.string().check(z.refine((v) => v.trim() === '' || pattern.test(v.trim()), message));
    validateStandardSchema(
      path,
      z.object({
        name: this.account()
          ? z.string().check(z.maxLength(120, t('crm.errors.tooLong')))
          : z
              .string()
              .check(
                z.trim(),
                z.minLength(1, t('crm.errors.nameRequired')),
                z.maxLength(120, t('crm.errors.tooLong')),
              ),
        phone: optional(PHONE, t('crm.errors.phone')),
        telegram: optional(TELEGRAM, t('crm.errors.telegram')),
        note: z.string().check(z.maxLength(4000, t('crm.errors.tooLong'))),
        tags: z.string(),
        source: z.string(),
      }),
    );
  });

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    const v = this.model();
    const orNull = (s: string): string | null => s.trim() || null;
    const body = {
      name: orNull(v.name),
      phone: orNull(v.phone),
      telegram: orNull(v.telegram),
      note: orNull(v.note),
      tags: v.tags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)
        .slice(0, 20),
      source: v.source || null,
    };
    await submit(this.clientForm, async () => {
      try {
        const current = this.client();
        const saved = await firstValueFrom(
          current
            ? this.http.put<CrmClient>(`/crm/clients/${current.id}`, body)
            : this.http.post<CrmClient>('/crm/clients', body),
        );
        this.saved.emit(saved);
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
