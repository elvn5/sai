import { injectLang } from '@/core/i18n/current-lang';
import { LANGS, LANG_NAMES, type Lang, isLang, splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { Menu, MenuPanel, MenuTrigger } from '@/ui/menu';
import { SiteLink } from '@/ui/site-link';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  computed,
  inject,
  input,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The language cookie, under the name the former Nuxt site (@nuxtjs/i18n)
// gave it, so visitors' saved choice keeps working.
const LANG_COOKIE = 'i18n_redirected';

// Opens the current page in another language. Search engines find the
// other languages through hreflang, so the menu only renders on demand;
// in the phone menu (showName) the languages are a row of links.
@Component({
  selector: 'app-language-switcher',
  imports: [Menu, MenuPanel, MenuTrigger, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  // A chosen language is remembered (the links carry hreflang).
  host: { '(click)': 'onClick($event)' },
  template: `
    @if (showName()) {
      <!-- The phone menu: every language at hand, no dropdown. -->
      <nav class="flex items-center gap-1" [attr.aria-label]="'language.switch' | transloco">
        <span class="icon-[heroicons--language] mr-0.5 size-[1.15rem] text-text-muted" aria-hidden="true"></span>
        @for (item of items(); track item.lang) {
          <a
            [appSiteLink]="item.path"
            [lang]="item.lang"
            [attr.hreflang]="item.lang"
            [attr.aria-current]="item.lang === lang() ? 'true' : null"
            [title]="item.name"
            class="rounded-md px-2 py-1.5 text-[0.8rem] font-medium text-text-muted aria-[current]:bg-brand-500/14 aria-[current]:text-brand-700 dark:aria-[current]:text-brand-300"
          >
            {{ item.lang.toUpperCase() }}
          </a>
        }
      </nav>
    } @else {
      <div appMenu #menu="appMenu" class="relative">
        <button
          type="button"
          appMenuTrigger
          class="flex h-9 items-center gap-1.5 rounded-[0.625rem] px-2 text-[0.8rem] font-medium text-text-muted transition-colors duration-(--dur-fast) hover:bg-text/5 hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500"
          [attr.aria-label]="'language.switch' | transloco"
          [title]="'language.switch' | transloco"
        >
          <span class="icon-[heroicons--language] size-[1.15rem]" aria-hidden="true"></span>
          <span>{{ lang().toUpperCase() }}</span>
        </button>
        @if (menu.open()) {
          <div appMenuPanel class="motion-drop-in absolute top-full right-0 z-50 mt-2 min-w-40 rounded-xl border border-border bg-surface p-1 text-text shadow-lg">
            @for (item of items(); track item.lang) {
              <a
                role="menuitem"
                [appSiteLink]="item.path"
                [lang]="item.lang"
                [attr.hreflang]="item.lang"
                [attr.aria-current]="item.lang === lang() ? 'true' : null"
                class="flex items-center justify-between gap-3 rounded-md px-3 py-2 text-sm hover:bg-bg focus:bg-bg focus:outline-none"
              >
                {{ item.name }}
                @if (item.lang === lang()) {
                  <span class="icon-[heroicons--check-20-solid] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
                }
              </a>
            }
          </div>
        }
      </div>
    }
  `,
})
export class LanguageSwitcher {
  // The phone menu: the languages as a row of links.
  readonly showName = input(false);
  protected readonly lang = injectLang();
  private readonly url = injectCurrentUrl();
  private readonly document = inject(DOCUMENT);

  protected readonly items = computed(() => {
    const [, query] = this.url().split(/\?(.*)/s, 2);
    const { path } = splitLang(this.url());
    const target = query ? `${path}?${query}` : path;
    return LANGS.map((lang) => ({ lang, name: LANG_NAMES[lang], path: target }));
  });

  protected onClick(event: MouseEvent): void {
    const lang = (event.target as Element).closest('a[hreflang]')?.getAttribute('hreflang');
    if (isLang(lang)) this.remember(lang);
  }

  private remember(lang: Lang): void {
    this.document.cookie = `${LANG_COOKIE}=${lang}; path=/; max-age=31536000; samesite=lax`;
  }
}
