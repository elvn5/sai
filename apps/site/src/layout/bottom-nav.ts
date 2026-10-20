import { AuthStore } from '@/core/auth/auth.store';
import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { SiteLink } from '@/ui/site-link';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  computed,
  inject,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Icon class names written out in full: Tailwind finds them in the source.
const ITEMS = [
  {
    path: '/',
    label: 'nav.feed',
    icon: 'icon-[heroicons--home]',
    active: 'icon-[heroicons--home-solid]',
  },
  {
    path: '/outings',
    label: 'nav.outings',
    icon: 'icon-[heroicons--user-group]',
    active: 'icon-[heroicons--user-group-solid]',
  },
  {
    path: '/tours',
    label: 'nav.tours',
    icon: 'icon-[heroicons--map]',
    active: 'icon-[heroicons--map-solid]',
  },
  {
    path: '/blog',
    label: 'nav.blog',
    icon: 'icon-[heroicons--newspaper]',
    active: 'icon-[heroicons--newspaper-solid]',
  },
] as const;

// Phones: the main sections under the thumb. The current one gets the
// solid icon and the brand color. It slides away while scrolling down and
// comes back on the way up.
@Component({
  selector: 'app-bottom-nav',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'lg:hidden' },
  template: `
    <nav
      class="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-border bg-surface/92 pb-[env(safe-area-inset-bottom)] backdrop-blur-md transition-transform duration-(--dur-base) ease-standard motion-reduce:transition-none"
      [class.translate-y-full]="hidden()"
      [attr.aria-label]="'nav.menu' | transloco"
    >
      @for (item of items; track item.path) {
        <a
          [appSiteLink]="item.path"
          class="flex h-16 flex-col items-center justify-center gap-0.5 text-[0.65rem] font-medium"
          [class]="current() === item.path ? 'font-semibold text-brand-700 dark:text-brand-300' : 'text-text-muted'"
          [attr.aria-current]="current() === item.path ? 'page' : null"
        >
          <span [class]="(current() === item.path ? item.active : item.icon) + ' size-6'" aria-hidden="true"></span>
          {{ item.label | transloco }}
        </a>
      }
      <a
        [appSiteLink]="auth.user() ? '/profile' : '/login'"
        class="flex h-16 flex-col items-center justify-center gap-0.5 text-[0.65rem] font-medium"
        [class]="current() === '/profile' ? 'font-semibold text-brand-700 dark:text-brand-300' : 'text-text-muted'"
        [attr.aria-current]="current() === '/profile' ? 'page' : null"
      >
        <span [class]="(current() === '/profile' ? 'icon-[heroicons--user-circle-solid]' : 'icon-[heroicons--user-circle]') + ' size-6'" aria-hidden="true"></span>
        {{ (auth.user() ? 'profile.title' : 'login.cta') | transloco }}
      </a>
    </nav>
  `,
})
export class BottomNav {
  protected readonly auth = inject(AuthStore);
  protected readonly items = ITEMS;
  protected readonly hidden = signal(false);
  private readonly url = injectCurrentUrl();

  protected readonly current = computed(() => {
    const path = splitLang(this.url()).path.split(/[?#]/)[0] ?? '/';
    if (path === '/') return '/';
    if (path.startsWith('/profile')) return '/profile';
    return ITEMS.find((item) => item.path !== '/' && path.startsWith(item.path))?.path;
  });

  constructor() {
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      let last = scrollY;
      let frame = 0;
      const onScroll = (): void => {
        if (frame) return;
        frame = requestAnimationFrame(() => {
          frame = 0;
          const y = scrollY;
          if (Math.abs(y - last) > 6) this.hidden.set(y > last && y > 120);
          last = y;
        });
      };
      addEventListener('scroll', onScroll, { passive: true });
      destroyRef.onDestroy(() => removeEventListener('scroll', onScroll));
    });
  }
}
