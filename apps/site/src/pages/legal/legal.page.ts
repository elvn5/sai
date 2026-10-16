import { injectLang } from '@/core/i18n/current-lang';
import { COMPANY } from '@/core/legal/company';
import { LEGAL, type LegalDocument } from '@/core/legal/documents';
import { formatDate } from '@/core/lib/format';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import type { LegalText } from '@/pages/legal/texts/legal-text';
import { OFFER } from '@/pages/legal/texts/offer';
import { PRIVACY } from '@/pages/legal/texts/privacy';
import { REFUNDS } from '@/pages/legal/texts/refunds';
import { LegalLinks } from '@/ui/legal-links';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const TEXTS: Record<LegalDocument, LegalText> = {
  offer: OFFER,
  privacy: PRIVACY,
  refunds: REFUNDS,
};

// /offer, /privacy, /refunds: a legal document (route data `doc`). The text
// is Russian in every language; drafts carry a notice and stay unindexed.
@Component({
  selector: 'app-legal-page',
  imports: [LegalLinks, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="mx-auto max-w-3xl px-4 py-12">
      <header>
        <h1 class="text-3xl font-extrabold text-text" lang="ru">{{ text().title }}</h1>
        @if (effectiveFrom(); as date) {
          <p class="mt-2 text-sm text-text-muted">{{ 'legal.effectiveFrom' | transloco: { date } }}</p>
        }
      </header>

      @if (draft()) {
        <p class="mt-6 flex gap-3 rounded-card border border-amber-400/60 bg-amber-400/10 p-4 text-sm text-text" role="note">
          <span class="icon-[heroicons--pencil-square] size-5 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden="true"></span>
          {{ 'legal.draft' | transloco }}
        </p>
      }
      @if (lang() !== 'ru') {
        <p class="mt-4 text-sm text-text-muted">{{ 'legal.russianOnly' | transloco }}</p>
      }

      <nav class="mt-8 rounded-card border border-border bg-surface p-4" [attr.aria-label]="'legal.contents' | transloco">
        <h2 class="text-sm font-semibold text-text">{{ 'legal.contents' | transloco }}</h2>
        <ol class="mt-2 list-decimal space-y-1 pl-5 text-sm" lang="ru">
          @for (section of text().sections; track section.id) {
            <li><a [href]="'#' + section.id" class="text-brand-700 hover:underline dark:text-brand-300">{{ section.title }}</a></li>
          }
          <li><a href="#requisites" class="text-brand-700 hover:underline dark:text-brand-300">Реквизиты</a></li>
        </ol>
      </nav>

      <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" lang="ru">
        @for (section of text().sections; track section.id; let index = $index) {
          <section [attr.aria-labelledby]="section.id">
            <h2 [id]="section.id" class="scroll-mt-24">{{ index + 1 }}. {{ section.title }}</h2>
            <div [innerHTML]="section.body"></div>
          </section>
        }
        <section aria-labelledby="requisites">
          <h2 id="requisites" class="scroll-mt-24">{{ text().sections.length + 1 }}. Реквизиты</h2>
          <dl class="not-prose grid grid-cols-1 gap-x-6 gap-y-2 text-text sm:grid-cols-[max-content_1fr]">
            @for (row of requisites; track row.label) {
              <dt class="font-semibold sm:font-normal sm:text-text-muted">{{ row.label }}</dt>
              <dd class="mb-2 sm:mb-0">
                @if (row.value) {
                  {{ row.value }}
                } @else {
                  <span class="legal-todo">[не указано]</span>
                }
              </dd>
            }
          </dl>
        </section>
      </div>

      <footer class="mt-12 border-t border-border pt-6">
        <h2 class="text-sm font-semibold text-text">{{ 'legal.otherDocuments' | transloco }}</h2>
        <app-legal-links class="mt-2 text-sm" [except]="doc()" linkClass="text-brand-700 hover:underline dark:text-brand-300" />
      </footer>
    </article>
  `,
})
export class LegalPage {
  readonly doc = input.required<LegalDocument>();

  protected readonly lang = injectLang();
  protected readonly text = computed(() => TEXTS[this.doc()]);
  protected readonly draft = computed(() => LEGAL[this.doc()].draft);
  protected readonly effectiveFrom = computed(() => {
    const date = this.text().effectiveFrom;
    return date
      ? formatDate(this.lang(), date, { day: 'numeric', month: 'long', year: 'numeric' })
      : null;
  });
  protected readonly requisites = [
    { label: 'Наименование', value: COMPANY.name },
    { label: 'ИНН', value: COMPANY.taxId },
    { label: 'Регистрационный номер', value: COMPANY.registrationNumber },
    { label: 'Адрес', value: COMPANY.address },
    { label: 'Email', value: COMPANY.email },
    { label: 'Телефон', value: COMPANY.phone },
  ];

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const text = this.text();
      return {
        title: text.title,
        description: text.description,
        robots: this.draft() ? 'noindex' : undefined,
        jsonLd: [seo.breadcrumbs([{ name: text.title, path: LEGAL[this.doc()].path }])],
      };
    });
  }
}
