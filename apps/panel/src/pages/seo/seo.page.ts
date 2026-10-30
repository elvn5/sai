import { LOCALE_OPTIONS, Locale, type PageSeo, SEO_PAGE_OPTIONS, SeoPage } from '@/core/api/types';
import { HistoryLog } from '@/ui/history-log';
import { ImageUploadField } from '@/ui/image-upload-field';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, effect, inject, signal } from '@angular/core';
import { FormField, form, submit } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

interface SeoForm {
  title: string;
  description: string;
  ogImageUrl: string | null;
}

// Per-page, per-language SEO overrides of the public site (services/content).
@Component({
  selector: 'app-seo-page',
  imports: [FormField, HistoryLog, ImageUploadField],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto max-w-2xl">
      <h1 class="mb-4 text-xl font-semibold text-text">SEO</h1>
      <div class="mb-4 grid grid-cols-2 gap-4">
        <label class="ui-field">
          Раздел сайта
          <select class="ui-input" [value]="page()" (change)="page.set($any($event.target).value)">
            @for (option of pageOptions; track option.value) {
              <option [value]="option.value">{{ option.label }}</option>
            }
          </select>
        </label>
        <label class="ui-field">
          Язык
          <select class="ui-input" [value]="locale()" (change)="locale.set($any($event.target).value)">
            @for (option of localeOptions; track option.value) {
              <option [value]="option.value">{{ option.label }}</option>
            }
          </select>
        </label>
      </div>

      @if (records.hasValue()) {
        <form class="ui-card flex flex-col gap-4 p-4" (submit)="save($event)" novalidate>
          <label class="ui-field">
            SEO заголовок (title)
            <input class="ui-input" [formField]="seoForm.title" />
            <span class="ui-hint">Показывается во вкладке браузера и в поиске</span>
          </label>
          <label class="ui-field">
            SEO описание (meta description)
            <textarea class="ui-input" rows="3" [formField]="seoForm.description"></textarea>
            <span class="ui-hint">Показывается в результатах поиска</span>
          </label>
          <div class="ui-field">
            Изображение для соцсетей (Open Graph)
            <app-image-upload-field [formField]="seoForm.ogImageUrl" />
          </div>
          <div class="flex justify-end">
            <button type="submit" class="ui-btn-primary" [disabled]="seoForm().submitting()">Сохранить</button>
          </div>
        </form>
      }
    </div>

    <app-history-log class="mx-auto mt-8 max-w-2xl" [modules]="['seo']" />
  `,
})
export class SeoSettingsPage {
  private readonly http = inject(HttpClient);
  protected readonly pageOptions = SEO_PAGE_OPTIONS;
  protected readonly localeOptions = LOCALE_OPTIONS;
  protected readonly page = signal<SeoPage>(SeoPage.HOME);
  protected readonly locale = signal<Locale>(Locale.RU);
  protected readonly records = httpResource<PageSeo[]>(() => '/page-seo/admin/all');

  protected readonly model = signal<SeoForm>({ title: '', description: '', ogImageUrl: null });
  protected readonly seoForm = form(this.model);

  constructor() {
    // Show the saved values of the selected page and language.
    effect(() => {
      const existing = this.records
        .value()
        ?.find((record) => record.page === this.page() && record.locale === this.locale());
      this.model.set({
        title: existing?.title ?? '',
        description: existing?.description ?? '',
        ogImageUrl: existing?.ogImageUrl ?? null,
      });
    });
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    await submit(this.seoForm, async () => {
      const value = this.model();
      await firstValueFrom(
        this.http.put(`/page-seo/${this.page()}/${this.locale()}`, {
          title: value.title || null,
          description: value.description || null,
          ogImageUrl: value.ogImageUrl,
        }),
      );
      this.records.reload();
      return undefined;
    });
  }
}
