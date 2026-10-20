import { AuthStore } from '@/core/auth/auth.store';
import { Notify } from '@/core/notify/notify';
import { ColorModeStore } from '@/core/theme/color-mode';
import { LanguageSwitcher } from '@/layout/language-switcher';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

export interface NavLink {
  path: string;
  label: string;
  icon: string;
}

// The header's menu panel on phones: the sections, language, mode and the
// account. Loaded on its own (layout/header.ts defers it).
@Component({
  selector: 'app-mobile-menu',
  imports: [LanguageSwitcher, PersonAvatar, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <nav class="motion-cascade flex flex-col gap-0.5" [attr.aria-label]="'footer.sections.nav' | transloco">
      <a
        appSiteLink="/"
        class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-3 font-medium"
        [class]="section() === '/' ? 'bg-brand-500/14 font-semibold text-brand-700 dark:text-brand-300' : ''"
        [attr.aria-current]="section() === '/' ? 'page' : null"
      >
        <span [class]="(section() === '/' ? 'icon-[heroicons--home-solid]' : 'icon-[heroicons--home] text-text-muted') + ' size-5'" aria-hidden="true"></span>
        {{ 'nav.feed' | transloco }}
      </a>
      @for (link of links(); track link.path) {
        <a
          [appSiteLink]="link.path"
          class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-3 font-medium"
          [class]="section() === link.path ? 'bg-brand-500/14 font-semibold text-brand-700 dark:text-brand-300' : ''"
          [attr.aria-current]="section() === link.path ? 'page' : null"
        >
          <span [class]="link.icon + ' size-5' + (section() === link.path ? '' : ' text-text-muted')" aria-hidden="true"></span>
          {{ link.label | transloco }}
        </a>
      }
    </nav>
    <div class="mt-1.5 flex items-center justify-between border-t border-border px-1 pt-3">
      <app-language-switcher [showName]="true" />
      <div class="ui-segment w-32" role="radiogroup" [attr.aria-label]="'theme.toggle' | transloco">
        <button type="button" role="radio" [attr.aria-checked]="colorMode.mode() === 'light'" [attr.aria-label]="'theme.light' | transloco" (click)="setMode('light', $event)">
          <span class="icon-[heroicons--sun] size-4 align-[-0.2em]" aria-hidden="true"></span>
        </button>
        <button type="button" role="radio" [attr.aria-checked]="colorMode.mode() === 'dark'" [attr.aria-label]="'theme.dark' | transloco" (click)="setMode('dark', $event)">
          <span class="icon-[heroicons--moon] size-4 align-[-0.2em]" aria-hidden="true"></span>
        </button>
      </div>
    </div>
    <div class="mt-3.5 flex flex-col gap-1">
      @if (auth.user(); as user) {
        <a appSiteLink="/profile" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 font-medium hover:bg-text/5">
          <app-person-avatar [person]="user" [size]="30" />{{ auth.displayName() }}
        </a>
        @if (auth.isPartner()) {
          <a appSiteLink="/partner" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 font-medium hover:bg-text/5">
            <span class="icon-[heroicons--squares-2x2] size-5 text-text-muted" aria-hidden="true"></span>{{ 'cabinet.title' | transloco }}
          </a>
        }
        <button type="button" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 text-left font-medium hover:bg-text/5" (click)="logout()">
          <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-5 text-text-muted" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
        </button>
      } @else {
        <a [appSiteLink]="loginLink()" class="ui-btn-primary ui-btn-lg w-full">
          <span class="icon-[heroicons--user-circle] size-[1.15rem]" aria-hidden="true"></span>{{ 'login.cta' | transloco }}
        </a>
      }
    </div>
  `,
})
export class MobileMenu {
  readonly links = input.required<readonly NavLink[]>();
  // The current section's path ("/tours"), as the header works it out.
  readonly section = input<string | undefined>();
  readonly loginLink = input.required<string>();
  // Signed out: the header closes the panel.
  readonly done = output();

  protected readonly auth = inject(AuthStore);
  protected readonly colorMode = inject(ColorModeStore);
  private readonly notify = inject(Notify);

  protected setMode(mode: 'light' | 'dark', event: MouseEvent): void {
    if (this.colorMode.mode() === mode) return;
    const origin = event.detail > 0 ? { x: event.clientX, y: event.clientY } : undefined;
    this.colorMode.toggle(origin);
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    this.done.emit();
    this.notify.info({ title: 'notify.signedOut' });
  }
}
