import type { DriverProfile } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { FieldErrors } from '@/ui/field-errors';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, linkedSignal } from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const PHONE = /^\+?[0-9\s\-()]{5,32}$/;

// /partner/driver-profile: the driver's card in the organizers' catalog —
// name, phone, where they work, a few words — and whether it shows there.
@Component({
  selector: 'app-driver-profile-page',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div>
        <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.driverProfile' | transloco }}</h1>
        <p class="text-sm text-text-muted">{{ 'driver.profile.subtitle' | transloco }}</p>
      </div>
      @if (profile.hasValue()) {
        <form class="ui-card flex flex-col gap-3 p-4" novalidate (submit)="save($event)">
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="ui-field">
              <label class="ui-label" for="driver-name">{{ 'crm.client.name' | transloco }}</label>
              <input id="driver-name" class="ui-input" autocomplete="name" [formField]="profileForm.name" />
              <app-field-errors [state]="profileForm.name()" />
            </div>
            <div class="ui-field">
              <label class="ui-label" for="driver-phone">{{ 'crm.client.phone' | transloco }}</label>
              <input id="driver-phone" class="ui-input" type="tel" autocomplete="tel" placeholder="+996 555 123 456" [formField]="profileForm.phone" />
              <app-field-errors [state]="profileForm.phone()" />
            </div>
          </div>
          <div class="ui-field">
            <label class="ui-label" for="driver-region">{{ 'driver.profile.region' | transloco }}</label>
            <input id="driver-region" class="ui-input" [placeholder]="'driver.profile.regionPlaceholder' | transloco" [formField]="profileForm.region" />
            <app-field-errors [state]="profileForm.region()" />
          </div>
          <div class="ui-field">
            <label class="ui-label" for="driver-about">{{ 'driver.profile.about' | transloco }}</label>
            <textarea id="driver-about" class="ui-input" rows="4" [placeholder]="'driver.profile.aboutPlaceholder' | transloco" [formField]="profileForm.about"></textarea>
            <app-field-errors [state]="profileForm.about()" />
          </div>
          <label class="flex items-start gap-2 text-sm text-text">
            <input type="checkbox" class="mt-0.5 size-4 accent-brand-600" [formField]="profileForm.published" />
            <span>{{ 'driver.profile.published' | transloco }}<span class="block text-xs text-text-muted">{{ 'driver.profile.publishedHint' | transloco }}</span></span>
          </label>
          <div>
            <button type="submit" class="ui-btn-primary" [disabled]="profileForm().submitting()">{{ 'crm.save' | transloco }}</button>
          </div>
        </form>
      }
    </div>
  `,
})
export class DriverProfilePage {
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly notify = inject(Notify);
  private readonly auth = inject(AuthStore);
  protected readonly profile = httpResource<DriverProfile>(() => '/driver/profile');

  private readonly model = linkedSignal(() => {
    const p = loaded(this.profile);
    return {
      name: p?.name || this.auth.displayName(),
      phone: p?.phone ?? this.auth.user()?.phone ?? '',
      region: p?.region ?? '',
      about: p?.about ?? '',
      published: p ? p.published || !p.name : true,
    };
  });

  protected readonly profileForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path,
      z.object({
        name: z
          .string()
          .check(
            z.trim(),
            z.minLength(1, t('crm.errors.nameRequired')),
            z.maxLength(120, t('crm.errors.tooLong')),
          ),
        phone: z
          .string()
          .check(z.refine((v) => v.trim() === '' || PHONE.test(v.trim()), t('crm.errors.phone'))),
        region: z.string().check(z.maxLength(120, t('crm.errors.tooLong'))),
        about: z.string().check(z.maxLength(2000, t('crm.errors.tooLong'))),
        published: z.boolean(),
      }),
    );
  });

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.driverProfile'),
      description: this.transloco.translate('driver.profile.subtitle'),
      robots: 'noindex',
    }));
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    const v = this.model();
    await submit(this.profileForm, async () => {
      try {
        await firstValueFrom(
          this.http.put<DriverProfile>('/driver/profile', {
            name: v.name.trim(),
            phone: v.phone.trim() || null,
            region: v.region.trim() || null,
            about: v.about.trim() || null,
            published: v.published,
          }),
        );
        this.profile.reload();
        this.notify.success({ title: 'driver.profile.saved' });
      } catch {
        this.notify.error({ title: 'crm.saveError' });
      }
      return undefined;
    });
  }
}
