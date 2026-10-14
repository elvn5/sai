import { injectLang } from '@/shared/i18n/current-lang';
import { LANGS, localizedPath } from '@/shared/i18n/langs';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { Meta, Title } from '@angular/platform-browser';
import { RouterLink } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// Placeholder home page of the migration's phase 0: proves SSR with the
// shared translations, the language prefixes, design tokens and icons.
// Replaced by the real home page in wave 3.
@Component({
  selector: 'app-home-page',
  imports: [RouterLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="mx-auto flex min-h-screen max-w-3xl flex-col justify-center gap-6 px-4 py-16">
      <p class="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-brand-600">
        <span class="icon-[heroicons--map] size-5" aria-hidden="true"></span>
        {{ 'brand.name' | transloco }}
      </p>
      <h1 class="text-4xl font-extrabold text-text">{{ 'home.tagline' | transloco }}</h1>
      <nav class="flex flex-wrap gap-2" [attr.aria-label]="'language.switch' | transloco">
        @for (item of languages(); track item.lang) {
          <a
            [routerLink]="item.path"
            class="rounded-card border border-border bg-surface px-4 py-2 text-sm font-medium"
            [class.text-brand-600]="item.lang === lang()"
            [attr.hreflang]="item.lang"
          >
            {{ item.lang.toUpperCase() }}
          </a>
        }
      </nav>
    </main>
  `,
})
export class HomePage {
  protected readonly lang = injectLang();
  protected readonly languages = computed(() =>
    LANGS.map((lang) => ({ lang, path: localizedPath(lang, '/') })),
  );

  constructor() {
    const transloco = inject(TranslocoService);
    const tagline = transloco.translate('home.tagline');
    inject(Title).setTitle(`${transloco.translate('brand.name')} — ${tagline}`);
    inject(Meta).updateTag({ name: 'description', content: tagline });
  }
}
