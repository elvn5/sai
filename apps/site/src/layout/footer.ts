import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { COMPANY } from '@/core/legal/company';
import { ColorThemeToggle } from '@/layout/color-theme-toggle';
import { AppLogo } from '@/ui/app-logo';
import { LegalLinks } from '@/ui/legal-links';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, Injector, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

@Component({
  selector: 'app-footer',
  host: { class: 'block' },
  imports: [AppLogo, ColorThemeToggle, LegalLinks, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <footer class="border-t border-border bg-surface text-text">
      <div class="ui-brandline"></div>
      <div class="ui-container grid gap-10 py-14 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <a appSiteLink="/" class="flex items-center gap-2 text-[1.06rem] font-extrabold tracking-tight">
            <span class="flex size-[1.875rem] items-center justify-center rounded-[0.5625rem] bg-brand-600 text-white">
              <app-logo class="size-[1.15rem]" />
            </span>
            {{ 'brand.name' | transloco }}
          </a>
          <p class="mt-4 max-w-xs text-sm text-text-muted">{{ 'home.tagline' | transloco }}</p>
          <div class="mt-6 flex items-center gap-3">
            @if (telegramUrl) {
              <a
                [href]="telegramUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="flex size-10 items-center justify-center rounded-full border border-border text-text-muted transition-colors hover:border-brand-500/50 hover:text-text"
                [attr.aria-label]="'footer.telegram' | transloco"
              >
                <span class="icon-[heroicons--paper-airplane] size-4" aria-hidden="true"></span>
              </a>
            }
            <a
              [appSiteLink]="auth.user() ? '/profile' : '/login'"
              class="flex size-10 items-center justify-center rounded-full border border-border text-text-muted transition-colors hover:border-brand-500/50 hover:text-text"
              [attr.aria-label]="'footer.account' | transloco"
            >
              <span class="icon-[heroicons--user] size-4" aria-hidden="true"></span>
            </a>
            <app-color-theme-toggle />
          </div>
        </div>

        <nav [attr.aria-label]="'footer.sections.nav' | transloco">
          <h2 class="text-xs font-semibold tracking-wider text-text-muted uppercase">{{ 'footer.sections.nav' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-text-muted">
            <li><a appSiteLink="/" class="hover:text-text">{{ 'nav.feed' | transloco }}</a></li>
            <li><a appSiteLink="/hikes" class="hover:text-text">{{ 'nav.hikes' | transloco }}</a></li>
            <li><a appSiteLink="/gallery" class="hover:text-text">{{ 'nav.gallery' | transloco }}</a></li>
            <li><a appSiteLink="/blog" class="hover:text-text">{{ 'nav.blog' | transloco }}</a></li>
          </ul>
        </nav>

        <div>
          <h2 class="text-xs font-semibold tracking-wider text-text-muted uppercase">{{ 'footer.sections.account' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-text-muted">
            <li>
              @if (auth.user()) {
                <a appSiteLink="/profile" class="hover:text-text">{{ 'footer.profile' | transloco }}</a>
              } @else {
                <a appSiteLink="/login" class="hover:text-text">{{ 'login.cta' | transloco }}</a>
              }
            </li>
            <li><a appSiteLink="/tours" class="hover:text-text">{{ 'footer.startTrip' | transloco }}</a></li>
          </ul>
        </div>

        <div>
          <h2 class="text-xs font-semibold tracking-wider text-text-muted uppercase">{{ 'footer.sections.contacts' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-text-muted">
            <li class="flex items-center gap-2">
              <span class="icon-[heroicons--map-pin] size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
              {{ 'footer.location' | transloco }}
            </li>
            @if (telegramUrl) {
              <li class="flex items-center gap-2">
                <span class="icon-[heroicons--paper-airplane] size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
                <a [href]="telegramUrl" target="_blank" rel="noopener noreferrer" class="hover:text-text">
                  &#64;{{ botUsername }}
                </a>
              </li>
            }
          </ul>
        </div>
      </div>

      <div class="border-t border-border">
        <div class="ui-container flex flex-col gap-3 py-5 text-center text-xs text-text-muted sm:flex-row sm:items-start sm:justify-between sm:text-left">
          <div>
            <p>
              {{ 'brand.name' | transloco }} — {{ 'footer.bottomTagline' | transloco }} © {{ year }}.
              {{ 'footer.rights' | transloco }}
            </p>
            @if (company.name) {
              <p class="mt-1">{{ requisites() }}</p>
            }
            @if (version) {
              <!-- A triple click opens the developer menu (layout/dev-menu.ts);
                   out of the tab order, it reads as plain text. -->
              <button type="button" tabindex="-1" class="mt-1 block cursor-text text-xs text-text-muted tabular-nums select-none" (click)="versionClick($event)">
                {{ 'footer.version' | transloco: { version } }}
              </button>
            }
          </div>
          <nav [attr.aria-label]="'footer.sections.legal' | transloco">
            <app-legal-links class="[&_ul]:justify-center sm:[&_ul]:justify-end" linkClass="text-text-muted hover:text-text hover:underline" />
          </nav>
        </div>
      </div>
    </footer>
  `,
})
export class Footer {
  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  protected readonly botUsername = inject(APP_CONFIG).telegramBotUsername;
  protected readonly telegramUrl = this.botUsername ? `https://t.me/${this.botUsername}` : null;
  protected readonly year = new Date().getFullYear();
  protected readonly company = COMPANY;
  protected readonly version = inject(APP_CONFIG).version;
  // The developer menu exists only with a password in the config.
  private readonly devMenu = Boolean(inject(APP_CONFIG).devMenuPasswordHash);
  private readonly injector = inject(Injector);

  protected async versionClick(event: MouseEvent): Promise<void> {
    if (event.detail !== 3 || !this.devMenu) return;
    // Loaded on demand, CDK Dialog included: not in every page's bundle.
    const { openDevMenu } = await import('@/layout/dev-menu');
    openDevMenu(this.injector);
  }

  // "ОсОО «…», ИНН 0123…, г. Бишкек, …" — the operator, as the bank asks.
  protected requisites(): string {
    const taxId = COMPANY.taxId && `${this.transloco.translate('footer.taxId')} ${COMPANY.taxId}`;
    return [COMPANY.name, taxId, COMPANY.address].filter(Boolean).join(', ');
  }
}
