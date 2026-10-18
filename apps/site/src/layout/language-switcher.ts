import { injectLang } from '@/core/i18n/current-lang';
import { LANGS, LANG_NAMES, type Lang, splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { SiteLink } from '@/ui/site-link';
import { CdkMenu, CdkMenuItem, CdkMenuTrigger } from '@angular/cdk/menu';
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
// other languages through hreflang, so the menu only renders on demand.
@Component({
  selector: 'app-language-switcher',
  imports: [CdkMenu, CdkMenuItem, CdkMenuTrigger, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button
      type="button"
      class="flex h-9 items-center gap-1.5 rounded-[0.625rem] px-2 text-[0.8rem] font-medium text-text-muted transition-colors duration-(--dur-fast) hover:bg-text/5 hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500"
      [cdkMenuTriggerFor]="menu"
      [attr.aria-label]="'language.switch' | transloco"
      [title]="'language.switch' | transloco"
    >
      <span class="icon-[heroicons--language] size-[1.15rem]" aria-hidden="true"></span>
      @if (showName()) {
        <span class="text-sm text-text">{{ currentName() }}</span>
      } @else {
        <span>{{ lang().toUpperCase() }}</span>
      }
    </button>
    <ng-template #menu>
      <div cdkMenu class="motion-drop-in min-w-40 rounded-xl border border-border bg-surface p-1 text-text shadow-lg">
        @for (item of items(); track item.lang) {
          <a
            cdkMenuItem
            [appSiteLink]="item.path"
            [lang]="item.lang"
            [attr.hreflang]="item.lang"
            [attr.aria-current]="item.lang === lang() ? 'true' : null"
            class="flex items-center justify-between gap-3 rounded-md px-3 py-2 text-sm hover:bg-bg focus:bg-bg focus:outline-none"
            (cdkMenuItemTriggered)="remember(item.lang)"
          >
            {{ item.name }}
            @if (item.lang === lang()) {
              <span class="icon-[heroicons--check-20-solid] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
            }
          </a>
        }
      </div>
    </ng-template>
  `,
})
export class LanguageSwitcher {
  // The language's full name next to the icon (the phone menu).
  readonly showName = input(false);
  protected readonly lang = injectLang();
  protected readonly currentName = computed(() => LANG_NAMES[this.lang()]);
  private readonly url = injectCurrentUrl();
  private readonly document = inject(DOCUMENT);

  protected readonly items = computed(() => {
    const [, query] = this.url().split(/\?(.*)/s, 2);
    const { path } = splitLang(this.url());
    const target = query ? `${path}?${query}` : path;
    return LANGS.map((lang) => ({ lang, name: LANG_NAMES[lang], path: target }));
  });

  protected remember(lang: Lang): void {
    this.document.cookie = `${LANG_COOKIE}=${lang}; path=/; max-age=31536000; samesite=lax`;
  }
}
