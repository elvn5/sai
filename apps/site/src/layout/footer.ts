import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { AppLogo } from '@/ui/app-logo';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

@Component({
  selector: 'app-footer',
  imports: [AppLogo, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <footer class="animated-gradient-border-top bg-linear-to-br from-brand-900 via-brand-700 to-brand-600 text-white">
      <div class="mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <a appSiteLink="/" class="flex items-center gap-2 text-lg font-bold">
            <app-logo class="size-7" />
            {{ 'brand.name' | transloco }}
          </a>
          <p class="mt-4 max-w-xs text-sm text-white/75">{{ 'home.tagline' | transloco }}</p>
          <div class="mt-6 flex items-center gap-3">
            @if (telegramUrl) {
              <a
                [href]="telegramUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="flex size-10 items-center justify-center rounded-full border border-white/25 transition-colors hover:border-white/60 hover:bg-white/10"
                [attr.aria-label]="'footer.telegram' | transloco"
              >
                <span class="icon-[heroicons--paper-airplane] size-4" aria-hidden="true"></span>
              </a>
            }
            <a
              [appSiteLink]="auth.user() ? '/profile' : '/login'"
              class="flex size-10 items-center justify-center rounded-full border border-white/25 transition-colors hover:border-white/60 hover:bg-white/10"
              [attr.aria-label]="'footer.account' | transloco"
            >
              <span class="icon-[heroicons--user] size-4" aria-hidden="true"></span>
            </a>
          </div>
        </div>

        <nav [attr.aria-label]="'footer.sections.nav' | transloco">
          <h2 class="text-xs font-semibold tracking-wider text-white/70 uppercase">{{ 'footer.sections.nav' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-white/85">
            <li><a appSiteLink="/" class="hover:text-white">{{ 'nav.feed' | transloco }}</a></li>
            <li><a appSiteLink="/guides" class="hover:text-white">{{ 'nav.guides' | transloco }}</a></li>
            <li><a appSiteLink="/gallery" class="hover:text-white">{{ 'nav.gallery' | transloco }}</a></li>
            <li><a appSiteLink="/blog" class="hover:text-white">{{ 'nav.blog' | transloco }}</a></li>
          </ul>
        </nav>

        <div>
          <h2 class="text-xs font-semibold tracking-wider text-white/70 uppercase">{{ 'footer.sections.account' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-white/85">
            <li>
              @if (auth.user()) {
                <a appSiteLink="/profile" class="hover:text-white">{{ 'footer.profile' | transloco }}</a>
              } @else {
                <a appSiteLink="/login" class="hover:text-white">{{ 'login.cta' | transloco }}</a>
              }
            </li>
            <li><a appSiteLink="/guides" class="hover:text-white">{{ 'footer.startTrip' | transloco }}</a></li>
          </ul>
        </div>

        <div>
          <h2 class="text-xs font-semibold tracking-wider text-white/70 uppercase">{{ 'footer.sections.contacts' | transloco }}</h2>
          <ul class="mt-4 space-y-3 text-sm text-white/85">
            <li class="flex items-center gap-2">
              <span class="icon-[heroicons--map-pin] size-4 shrink-0 text-white/60" aria-hidden="true"></span>
              {{ 'footer.location' | transloco }}
            </li>
            @if (telegramUrl) {
              <li class="flex items-center gap-2">
                <span class="icon-[heroicons--paper-airplane] size-4 shrink-0 text-white/60" aria-hidden="true"></span>
                <a [href]="telegramUrl" target="_blank" rel="noopener noreferrer" class="hover:text-white">
                  &#64;{{ botUsername }}
                </a>
              </li>
            }
          </ul>
        </div>
      </div>

      <div class="border-t border-white/15">
        <p class="mx-auto max-w-6xl px-4 py-5 text-center text-xs text-white/75 sm:text-left">
          {{ 'brand.name' | transloco }} — {{ 'footer.bottomTagline' | transloco }} © {{ year }}.
          {{ 'footer.rights' | transloco }}
        </p>
      </div>
    </footer>
  `,
})
export class Footer {
  protected readonly auth = inject(AuthStore);
  protected readonly botUsername = inject(APP_CONFIG).telegramBotUsername;
  protected readonly telegramUrl = this.botUsername ? `https://t.me/${this.botUsername}` : null;
  protected readonly year = new Date().getFullYear();
}
