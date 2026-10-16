import {
  type Achievement,
  type HallOfFame,
  type MyAchievements,
  type Order,
  type Outing,
  PartnerStatus,
} from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import { AchievementShelf } from '@/ui/achievements/achievement-shelf';
import { AwardCodeCard } from '@/ui/achievements/award-code-card';
import { HallPodium } from '@/ui/achievements/hall-podium';
import { OrderCard } from '@/ui/order-card';
import { OutingCard } from '@/ui/outing-card';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { TelegramLink } from '@/ui/telegram-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const TABS = [
  { key: 'info', icon: 'icon-[heroicons--user-circle]' },
  { key: 'achievements', icon: 'icon-[mdi--trophy-outline]' },
  { key: 'bookings', icon: 'icon-[mdi--ticket-confirmation-outline]' },
  { key: 'history', icon: 'icon-[heroicons--map]' },
  { key: 'partner', icon: 'icon-[heroicons--briefcase]' },
] as const;

type Tab = (typeof TABS)[number]['key'];

// /profile: the account, the hall of fame (own medals, award code, the
// leaders), bookings, outings and the partner application. ?tab=partner (etc.) opens a tab. Browser only.
@Component({
  selector: 'app-profile-page',
  imports: [
    AchievementShelf,
    AwardCodeCard,
    HallPodium,
    OrderCard,
    OutingCard,
    PersonAvatar,
    SiteLink,
    TelegramLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (auth.user(); as user) {
      <section class="mx-auto max-w-3xl px-4 py-12">
        <div class="flex items-center gap-4">
          <app-person-avatar [person]="user" [size]="64" />
          <div>
            <h1 class="text-2xl font-extrabold text-text">{{ displayName() || ('profile.title' | transloco) }}</h1>
            @if (user.username) {
              <p class="text-sm text-text-muted">&#64;{{ user.username }}</p>
            }
          </div>
        </div>

        <div class="-mx-4 mt-8 overflow-x-auto px-4" >
          <div class="flex min-w-max gap-1 rounded-lg border border-border bg-surface p-1" role="tablist" [attr.aria-label]="'profile.title' | transloco">
            @for (item of tabs; track item.key) {
              <button
                type="button"
                role="tab"
                class="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors"
                [class]="active() === item.key ? 'bg-brand-600 text-white' : 'text-text-muted hover:text-text'"
                [id]="'tab-' + item.key"
                [attr.aria-selected]="active() === item.key"
                [attr.aria-controls]="'panel-' + item.key"
                [attr.tabindex]="active() === item.key ? 0 : -1"
                (click)="open(item.key)"
                (keydown)="onKeydown($event)"
              >
                <span [class]="item.icon + ' size-5'" aria-hidden="true"></span>{{ 'profile.tabs.' + item.key | transloco }}
              </button>
            }
          </div>
        </div>

        <div class="mt-6" role="tabpanel" [id]="'panel-' + active()" [attr.aria-labelledby]="'tab-' + active()">
          @switch (active()) {
            @case ('info') {
              <dl class="flex flex-col gap-4">
                <div class="ui-card flex items-center justify-between px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.email' | transloco }}</dt>
                  <dd class="text-sm font-medium text-text">{{ user.email ?? '—' }}</dd>
                </div>
                <div class="ui-card flex items-center justify-between px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.phone' | transloco }}</dt>
                  <dd class="text-sm font-medium text-text">{{ user.phone ?? '—' }}</dd>
                </div>
                <div class="ui-card flex flex-col gap-2 px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.telegram' | transloco }}</dt>
                  <dd><app-telegram-link /></dd>
                </div>
              </dl>
              <button type="button" class="ui-btn-soft mt-6" (click)="logout()">{{ 'profile.logout' | transloco }}</button>
            }
            @case ('achievements') {
              <div class="flex flex-col gap-6">
                @if (achievementsValue(); as mine) {
                  <dl class="grid grid-cols-3 gap-3">
                    <div class="ui-card flex flex-col-reverse px-3 py-2 text-center">
                      <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.points' | transloco }}</dt>
                      <dd class="text-2xl font-black text-amber-600 dark:text-amber-400">{{ mine.points }}</dd>
                    </div>
                    <div class="ui-card flex flex-col-reverse px-3 py-2 text-center">
                      <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.earned' | transloco }}</dt>
                      <dd class="text-2xl font-black text-text">{{ mine.awards.length }}</dd>
                    </div>
                    <div class="ui-card flex flex-col-reverse px-3 py-2 text-center">
                      <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.rank' | transloco }}</dt>
                      <dd class="text-2xl font-black text-brand-700 dark:text-brand-300">{{ myRank() ? '#' + myRank() : '—' }}</dd>
                    </div>
                  </dl>
                  <section class="ui-card p-5" aria-labelledby="profile-medals">
                    <h2 id="profile-medals" class="mb-4 font-semibold text-text">{{ 'hallOfFame.mine.heading' | transloco }}</h2>
                    <app-achievement-shelf [mine]="mine" [catalog]="catalogList()" />
                  </section>
                  <app-award-code-card [code]="mine.code" />
                  <p class="text-sm text-text-muted">{{ 'hallOfFame.mine.howTo' | transloco }}</p>
                }
                @if (leaders().length > 0) {
                  <section aria-labelledby="profile-leaders">
                    <h2 id="profile-leaders" class="mb-8 font-semibold text-text">{{ 'hallOfFame.leaders.heading' | transloco }}</h2>
                    <app-hall-podium [leaders]="leaders()" />
                  </section>
                }
                <a appSiteLink="/hall-of-fame" class="ui-btn-soft self-start">
                  <span class="icon-[mdi--trophy] size-4" aria-hidden="true"></span>{{ 'profile.achievements.open' | transloco }}
                </a>
              </div>
            }
            @case ('bookings') {
              @if (orderList().length > 0) {
                <ul class="flex flex-col gap-2">
                  @for (order of orderList(); track order.id) {
                    <li><app-order-card [order]="order" /></li>
                  }
                </ul>
              } @else if (orders.hasValue()) {
                <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
                  <span class="icon-[mdi--ticket-confirmation-outline] size-10 text-text-muted" aria-hidden="true"></span>
                  <p class="font-medium text-text">{{ 'booking.noBookings' | transloco }}</p>
                  <a appSiteLink="/tours" class="ui-btn-primary mt-2">{{ 'booking.browseTours' | transloco }}</a>
                </div>
              }
            }
            @case ('history') {
              <div class="flex flex-col gap-8">
                @if (upcoming().length > 0) {
                  <section aria-labelledby="history-upcoming">
                    <h2 id="history-upcoming" class="mb-3 font-semibold text-text">{{ 'profile.history.upcoming' | transloco }}</h2>
                    <ul class="flex flex-col gap-2">
                      @for (outing of upcoming(); track outing.id) {
                        <li><app-outing-card [outing]="outing" /></li>
                      }
                    </ul>
                  </section>
                }
                @if (past().length > 0) {
                  <section aria-labelledby="history-past">
                    <h2 id="history-past" class="mb-3 font-semibold text-text">{{ 'profile.history.past' | transloco }}</h2>
                    <ul class="flex flex-col gap-2">
                      @for (outing of past(); track outing.id) {
                        <li><app-outing-card [outing]="outing" /></li>
                      }
                    </ul>
                  </section>
                }
                @if (outings.hasValue() && upcoming().length + past().length === 0) {
                  <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
                    <span class="icon-[heroicons--map] size-10 text-text-muted" aria-hidden="true"></span>
                    <p class="font-medium text-text">{{ 'profile.history.empty' | transloco }}</p>
                    <p class="max-w-sm text-sm text-text-muted">{{ 'profile.history.emptyHint' | transloco }}</p>
                    <a appSiteLink="/outings" class="ui-btn-primary mt-2">{{ 'profile.history.cta' | transloco }}</a>
                  </div>
                }
              </div>
            }
            @case ('partner') {
              <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
                @switch (user.partnerStatus) {
                  @case (statuses.APPROVED) {
                    <span class="icon-[heroicons--check-badge] size-10 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
                    <p class="font-medium text-text">{{ 'profile.partner.approved' | transloco }}</p>
                    <div class="mt-2 flex flex-wrap justify-center gap-2">
                      <a appSiteLink="/partner" class="ui-btn-primary"><span class="icon-[heroicons--squares-2x2] size-4" aria-hidden="true"></span>{{ 'cabinet.open' | transloco }}</a>
                      <a appSiteLink="/achievements/award" class="ui-btn-soft"><span class="icon-[mdi--medal] size-4" aria-hidden="true"></span>{{ 'profile.partner.awardCta' | transloco }}</a>
                      <a appSiteLink="/tours/new" class="ui-btn-soft"><span class="icon-[heroicons--map] size-4" aria-hidden="true"></span>{{ 'profile.partner.addTourCta' | transloco }}</a>
                      <a appSiteLink="/blog/new" class="ui-btn-soft"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'blog.form.newTitle' | transloco }}</a>
                    </div>
                  }
                  @case (statuses.PENDING) {
                    <span class="icon-[heroicons--clock] size-10 text-text-muted" aria-hidden="true"></span>
                    <p class="font-medium text-text">{{ 'profile.partner.pending' | transloco }}</p>
                  }
                  @default {
                    <span class="icon-[heroicons--briefcase] size-10 text-text-muted" aria-hidden="true"></span>
                    @if (user.partnerStatus === statuses.REJECTED) {
                      <p class="font-medium text-text">{{ 'profile.partner.rejected' | transloco }}</p>
                    }
                    <p class="max-w-sm text-sm text-text-muted">{{ 'profile.partner.pitch' | transloco }}</p>
                    <p class="max-w-sm text-sm text-text-muted">{{ 'profile.partner.publishOnly' | transloco }}</p>
                    <button type="button" class="ui-btn-primary mt-2" [disabled]="applying()" (click)="apply()">{{ 'profile.partner.applyCta' | transloco }}</button>
                    @if (applyFailed()) {
                      <p class="ui-error" role="alert">{{ 'outings.errors.generic' | transloco }}</p>
                    }
                  }
                }
              </div>
            }
          }
        </div>
      </section>
    }
  `,
})
export class ProfilePage {
  readonly tab = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly router = inject(Router);
  private readonly navigate = injectSiteNavigate();
  protected readonly tabs = TABS;
  protected readonly statuses = PartnerStatus;

  protected readonly active = computed(
    (): Tab => TABS.find((item) => item.key === this.tab())?.key ?? 'info',
  );
  protected readonly displayName = computed(() => {
    const user = this.auth.user();
    return user?.firstName ?? user?.username ?? user?.email ?? '';
  });

  // Each tab's data loads when the tab opens.
  protected readonly orders = httpResource<Order[]>(() =>
    this.active() === 'bookings' ? '/orders/mine' : undefined,
  );
  protected readonly outings = httpResource<Outing[]>(() =>
    this.active() === 'history' ? '/outings/mine' : undefined,
  );
  private readonly achievements = httpResource<MyAchievements>(() =>
    this.active() === 'achievements' ? '/achievements/me' : undefined,
  );
  private readonly hall = httpResource<HallOfFame>(() =>
    this.active() === 'achievements' ? '/hall-of-fame' : undefined,
  );
  private readonly catalog = httpResource<Achievement[]>(() =>
    this.active() === 'achievements' ? '/achievements' : undefined,
  );
  protected readonly orderList = computed(() => loaded(this.orders) ?? []);
  protected readonly achievementsValue = computed(() => loaded(this.achievements));
  protected readonly catalogList = computed(() => loaded(this.catalog) ?? []);
  protected readonly leaders = computed(() => loaded(this.hall)?.leaders ?? []);
  protected readonly myRank = computed(
    () => this.leaders().find((entry) => entry.user.id === this.auth.user()?.id)?.rank ?? null,
  );
  protected readonly upcoming = computed(() =>
    (loaded(this.outings) ?? [])
      .filter((outing) => new Date(outing.startsAt).getTime() > Date.now())
      .reverse(),
  );
  protected readonly past = computed(() =>
    (loaded(this.outings) ?? []).filter(
      (outing) => new Date(outing.startsAt).getTime() <= Date.now(),
    ),
  );
  protected readonly applying = signal(false);
  protected readonly applyFailed = signal(false);

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('profile.title'),
      description: transloco.translate('login.subtitle'),
      robots: 'noindex',
    }));
  }

  protected open(tab: Tab): void {
    void this.router.navigate([], {
      queryParams: { tab: tab === 'info' ? null : tab },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }

  // Arrow keys move between tabs (WAI-ARIA tabs pattern).
  protected onKeydown(event: KeyboardEvent): void {
    const step = { ArrowRight: 1, ArrowLeft: -1 }[event.key];
    if (!step) return;
    event.preventDefault();
    const index = TABS.findIndex((item) => item.key === this.active());
    const next = TABS[(index + step + TABS.length) % TABS.length];
    if (!next) return;
    this.open(next.key);
    (event.currentTarget as HTMLElement).ownerDocument.getElementById(`tab-${next.key}`)?.focus();
  }

  protected async apply(): Promise<void> {
    this.applying.set(true);
    this.applyFailed.set(false);
    try {
      await this.auth.applyAsPartner();
    } catch {
      this.applyFailed.set(true);
    } finally {
      this.applying.set(false);
    }
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.navigate('/');
  }
}
