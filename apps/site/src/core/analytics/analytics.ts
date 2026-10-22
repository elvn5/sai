import { CookieConsentStore } from '@/core/analytics/consent';
import { type AnalyticsBackend, loadFirebaseAnalytics } from '@/core/analytics/firebase';
import { APP_CONFIG, type FirebaseWebConfig } from '@/core/config/app-config';
import { splitLang } from '@/core/i18n/langs';
import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import {
  ApplicationRef,
  Injectable,
  InjectionToken,
  Injector,
  PLATFORM_ID,
  type Signal,
  afterNextRender,
  effect,
  inject,
  signal,
  untracked,
} from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { TranslocoService } from '@jsverse/transloco';
import { filter } from 'rxjs';

// The site's events (Google Analytics 4 through Firebase). Each also
// carries the page's language ("lang"); the bot (services/bot) reports its
// own under the same names where they mean the same thing.
export interface AnalyticsEvents {
  view_tour: { tour_id: string };
  view_outing: { outing_id: string };
  join_outing: { outing_id: string };
  leave_outing: { outing_id: string };
  // Online booking is paused: contacting the organizer is the booking.
  contact_organizer: { item_type: string; item_id: string; method: string };
  login: { method: string };
  sign_up: { method: string };
}

export type AnalyticsLoader = (config: FirebaseWebConfig) => Promise<AnalyticsBackend | null>;

// Overridable in tests.
export const ANALYTICS_LOADER = new InjectionToken<AnalyticsLoader>('ANALYTICS_LOADER', {
  providedIn: 'root',
  factory: () => loadFirebaseAnalytics,
});

// Firebase Analytics, in the browser only and only once the visitor allows
// analytics cookies (core/analytics/consent.ts); off without the Firebase
// config (APP_CONFIG.firebase). Nothing runs on the server.
@Injectable({ providedIn: 'root' })
export class Analytics {
  private readonly config = inject(APP_CONFIG).firebase;
  private readonly consent = inject(CookieConsentStore);
  private readonly loader = inject(ANALYTICS_LOADER);
  private readonly router = inject(Router);
  private readonly appRef = inject(ApplicationRef);
  private readonly document = inject(DOCUMENT);
  private readonly transloco = inject(TranslocoService);
  private readonly injector = inject(Injector);
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID));
  private backend: Promise<AnalyticsBackend | null> | null = null;
  private started = false;
  private lastPage: string | null = null;

  // Whether the site has analytics to ask consent for.
  readonly available = this.config !== null;

  // Follows the visitor's consent and the pages they open. Browser only,
  // once (the root component, after the first render).
  init(): void {
    if (!this.browser || !this.config || this.started) return;
    this.started = true;
    this.consent.init();
    effect(
      () => {
        const allowed = this.consent.analyticsAllowed();
        untracked(() => (allowed ? this.enable() : this.disable()));
      },
      { injector: this.injector },
    );
    this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe(() => void this.pageView());
  }

  track<K extends keyof AnalyticsEvents>(name: K, params: AnalyticsEvents[K]): void {
    if (!this.browser || !this.config) return;
    // A page may report before the root component has started analytics.
    this.consent.init();
    if (!this.consent.analyticsAllowed()) return;
    void this.send(name, params);
  }

  private enable(): void {
    // Allowed again after a refusal in this visit.
    void this.backend?.then((backend) => backend?.setEnabled(true));
    this.lastPage = null;
    void this.pageView();
  }

  private load(): Promise<AnalyticsBackend | null> {
    const config = this.config;
    if (!config) return Promise.resolve(null);
    this.backend ??= this.loader(config).catch(() => null);
    return this.backend;
  }

  private disable(): void {
    void this.backend?.then((backend) => backend?.setEnabled(false));
    if (this.consent.choice() === 'necessary') this.dropCookies();
  }

  // A page view once the page has its data and title (the app is stable).
  private async pageView(): Promise<void> {
    if (!this.consent.analyticsAllowed()) return;
    const url = this.router.url;
    if (url === this.lastPage) return;
    this.lastPage = url;
    await this.appRef.whenStable();
    if (url !== this.router.url) return;
    await this.send('page_view', {
      page_location: this.document.location.href,
      page_title: this.document.title,
      // The same page in every language, for reports across them.
      content_path: splitLang(url).path,
    });
  }

  private async send(name: string, params: object): Promise<void> {
    const backend = await this.load();
    backend?.log(name, { ...params, lang: this.transloco.getActiveLang() });
  }

  // Google Analytics' cookies (_ga, _ga_<id>), on this host and its parents.
  private dropCookies(): void {
    const host = this.document.location.hostname;
    const parts = host.split('.');
    const domains = parts.map((_, i) => parts.slice(i).join('.')).filter((d) => d.includes('.'));
    for (const cookie of this.document.cookie.split(';')) {
      const name = cookie.split('=')[0]?.trim() ?? '';
      if (!name.startsWith('_ga')) continue;
      const expired = `${name}=; Max-Age=0; path=/`;
      this.document.cookie = expired;
      for (const domain of domains) this.document.cookie = `${expired}; domain=.${domain}`;
    }
  }
}

// Reports one view of a tour or an outing, once it's shown in the browser
// (again when the page moves to another one). Call in an injection context.
export function trackItemView(
  name: 'view_tour' | 'view_outing',
  id: Signal<string | undefined>,
): void {
  const analytics = inject(Analytics);
  const shown = signal(false);
  let sent: string | undefined;
  afterNextRender(() => shown.set(true));
  effect(() => {
    const current = id();
    if (!shown() || !current || current === sent) return;
    sent = current;
    untracked(() =>
      name === 'view_tour'
        ? analytics.track(name, { tour_id: current })
        : analytics.track(name, { outing_id: current }),
    );
  });
}
