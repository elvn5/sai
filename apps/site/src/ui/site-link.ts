import { injectLang } from '@/core/i18n/current-lang';
import { type Lang, localizedPath } from '@/core/i18n/langs';
import { Directive, computed, inject, input } from '@angular/core';
import { Router } from '@angular/router';

// A link to a site page by its unprefixed path ("/tours", "/tours?page=2"),
// in the current language (or `lang`), opened inside the app.
@Directive({
  selector: 'a[appSiteLink]',
  host: { '[attr.href]': 'href()', '(click)': 'navigate($event)' },
})
export class SiteLink {
  readonly appSiteLink = input.required<string>();
  readonly lang = input<Lang | undefined>(undefined);
  private readonly currentLang = injectLang();
  private readonly router = inject(Router);

  protected readonly href = computed(() => {
    const [path = '/', query] = this.appSiteLink().split(/\?(.*)/s, 2);
    const localized = localizedPath(this.lang() ?? this.currentLang(), path);
    return query ? `${localized}?${query}` : localized;
  });

  protected navigate(event: MouseEvent): void {
    // New tab or window, downloads: the browser's business.
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey)
      return;
    event.preventDefault();
    void this.router.navigateByUrl(this.href());
  }
}
