import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Catalog pagination: a link back to the previous page, and "show more",
// which is a real link to the next page for crawlers but loads it in place
// for people.
@Component({
  selector: 'app-pager',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (previous() || next()) {
      <nav class="mt-8 flex items-center justify-center gap-3" [attr.aria-label]="'pagination.more' | transloco">
        @if (previous(); as link) {
          <a class="ui-btn-ghost" [appSiteLink]="link" rel="prev">
            <span class="icon-[heroicons--chevron-left] size-4" aria-hidden="true"></span>
            {{ 'pagination.previous' | transloco }}
          </a>
        }
        @if (next(); as link) {
          <a class="ui-btn-soft" [href]="localized(link)" rel="next" [attr.aria-busy]="loading()" (click)="more($event)">
            {{ (loading() ? 'gallery.loadingMore' : 'listing.loadMore') | transloco }}
          </a>
        }
      </nav>
    }
  `,
})
export class Pager {
  // Unprefixed links ("/hotels?page=2"), or null when there's no such page.
  readonly previous = input<string | null>(null);
  readonly next = input<string | null>(null);
  readonly loading = input(false);
  readonly loadMore = output();
  private readonly lang = injectLang();
  protected readonly localized = (path: string): string => localizedPath(this.lang(), path);

  protected more(event: MouseEvent): void {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey) return;
    event.preventDefault();
    this.loadMore.emit();
  }
}
