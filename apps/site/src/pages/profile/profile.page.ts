import {
  type Achievement,
  type BlogPost,
  type HallOfFame,
  type MyAchievements,
  type PartnerKind,
  PartnerStatus,
  type Tour,
} from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { FavoritesStore } from '@/core/favorites/favorites.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { ParticipantFormCard } from '@/pages/profile/participant-form-card';
import { PasswordCard } from '@/pages/profile/password-card';
import { AchievementShelf } from '@/ui/achievements/achievement-shelf';
import { AwardCodeCard } from '@/ui/achievements/award-code-card';
import { HallPodium } from '@/ui/achievements/hall-podium';
import { ListSkeleton } from '@/ui/list-skeleton';
import { TourCard } from '@/ui/listing/tour-card';
import { LoadError } from '@/ui/load-error';
import { MountainScene } from '@/ui/mountain-scene';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { TelegramLink } from '@/ui/telegram-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const TABS = [
  { key: 'info', icon: 'icon-[heroicons--user]' },
  { key: 'favorites', icon: 'icon-[heroicons--heart]' },
  { key: 'history', icon: 'icon-[heroicons--flag]' },
  { key: 'listings', icon: 'icon-[heroicons--rectangle-stack]' },
  { key: 'achievements', icon: 'icon-[heroicons--trophy]' },
  { key: 'partner', icon: 'icon-[heroicons--briefcase]' },
] as const;

type Tab = (typeof TABS)[number]['key'];

// What /favorites/mine answers: the saved listings that are still published
// (only tours are shown while hotels and local guides are paused).
interface Favorites {
  tours: Tour[];
}

interface ListingItem {
  key: string;
  // Translation key of the kind ("Tour", "Post").
  kind: string;
  icon: string;
  title: string;
  path: string;
  editPath: string | null;
  draft: boolean;
}

// /profile: the account, the hall of fame (own medals, award code, the
// leaders), the tours signed up for and the partner application. ?tab=partner (etc.) opens a tab. Browser only.
@Component({
  selector: 'app-profile-page',
  imports: [
    AchievementShelf,
    AwardCodeCard,
    HallPodium,
    MountainScene,
    ListSkeleton,
    LoadError,
    ParticipantFormCard,
    PasswordCard,
    PersonAvatar,
    SiteLink,
    StateBlock,
    TelegramLink,
    TourCard,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    /* Tabs: pills on phones, an underline on wider screens. */
    .tab {
      position: relative;
      display: inline-flex;
      flex-shrink: 0;
      align-items: center;
      gap: 0.375rem;
      white-space: nowrap;
      font-weight: 500;
      color: var(--app-text-muted);
      transition: color var(--dur-fast) var(--ease-standard), background-color var(--dur-fast) var(--ease-standard), border-color var(--dur-fast) var(--ease-standard);
    }
    .tab:hover {
      color: var(--app-text);
    }
    @media (width < 40rem) {
      .tab {
        border: 1px solid var(--app-border);
        border-radius: 9999px;
        padding: 0.3rem 0.75rem;
        font-size: 0.75rem;
        background: var(--app-surface);
      }
      .tab [class*="icon-["] {
        display: none;
      }
      .tab[aria-selected="true"] {
        border-color: var(--brand-500);
        background: color-mix(in oklab, var(--brand-500) 10%, transparent);
        color: var(--brand-700);
      }
      :host-context(.dark) .tab[aria-selected="true"] {
        color: var(--brand-300);
      }
    }
    @media (width >= 40rem) {
      .tabs {
        border-bottom: 1px solid var(--app-border);
      }
      .tab {
        padding: 0.625rem 0.75rem;
        font-size: 0.875rem;
      }
      .tab::after {
        content: "";
        position: absolute;
        inset: auto 0.5rem -1px;
        height: 2px;
        border-radius: 9999px;
        background: var(--brand-500);
        transform: scaleX(0);
        transition: transform var(--dur-base) var(--ease-spring);
      }
      .tab[aria-selected="true"] {
        color: var(--app-text);
      }
      .tab[aria-selected="true"]::after {
        transform: scaleX(1);
      }
    }
  `,
  template: `
    @if (auth.user(); as user) {
      <div class="relative isolate h-24 overflow-hidden sm:h-28">
        <app-mountain-scene palette="dusk" [lake]="false" />
        <div class="absolute inset-0 -z-10 bg-linear-to-b from-black/20 to-black/45"></div>
      </div>
      <section class="ui-container pb-14">
        <div class="relative -mt-10 flex flex-wrap items-start gap-4">
          <app-person-avatar class="ring-4 ring-bg" [person]="user" [size]="80" />
          <div class="min-w-0 flex-1 pt-11">
            <h1 class="truncate text-xl font-extrabold tracking-tight text-text sm:text-[1.625rem]">{{ displayName() || ('profile.title' | transloco) }}</h1>
            @if (user.username) {
              <p class="text-sm text-text-muted">&#64;{{ user.username }}</p>
            }
          </div>
          <button type="button" class="ui-btn-soft mt-12 max-sm:hidden" (click)="logout()">
            <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-4" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
          </button>
        </div>

        <dl class="mt-5 grid grid-cols-3 gap-2.5 sm:gap-3">
          @for (stat of stats(); track stat.key) {
            <div class="ui-card px-4 py-3">
              <dt class="text-xs text-text-muted">{{ 'profile.stats.' + stat.key | transloco }}</dt>
              <dd class="flex items-center gap-1.5 text-[1.375rem] font-extrabold text-text tabular-nums">
                @if (stat.value === null) {
                  <span class="ui-skeleton inline-block h-6 w-10"></span>
                } @else {
                  <span class="motion-roll-in">{{ stat.value }}</span>
                }
                @if (stat.key === 'awards') {
                  <span class="icon-[heroicons--trophy-solid] size-5 text-secondary-400" aria-hidden="true"></span>
                }
              </dd>
            </div>
          }
        </dl>

        <div class="tabs -mx-4 mt-5 flex gap-1.5 overflow-x-auto px-4 [mask-image:linear-gradient(to_right,#000_85%,transparent)] sm:mx-0 sm:gap-1 sm:px-0 sm:[mask-image:none]" role="tablist" [attr.aria-label]="'profile.title' | transloco">
          @for (item of tabs; track item.key) {
            <button
              type="button"
              role="tab"
              class="tab"
              [id]="'tab-' + item.key"
              [attr.aria-selected]="active() === item.key"
              [attr.aria-controls]="'panel-' + item.key"
              [attr.tabindex]="active() === item.key ? 0 : -1"
              (click)="open(item.key)"
              (keydown)="onKeydown($event)"
            >
              <span [class]="item.icon + ' size-[1.1rem]'" aria-hidden="true"></span>{{ 'profile.tabs.' + item.key | transloco }}
            </button>
          }
        </div>

        <div class="mt-5" role="tabpanel" [id]="'panel-' + active()" [attr.aria-labelledby]="'tab-' + active()">
          @switch (active()) {
            @case ('info') {
              <div class="flex flex-col gap-4">
              <dl class="ui-card flex flex-col divide-y divide-border">
                <div class="flex items-center justify-between px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.email' | transloco }}</dt>
                  <dd class="text-sm font-medium text-text">{{ user.email ?? '—' }}</dd>
                </div>
                <div class="flex items-center justify-between px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.phone' | transloco }}</dt>
                  <dd class="text-sm font-medium text-text">{{ user.phone ?? '—' }}</dd>
                </div>
                <div class="flex flex-col gap-2 px-4 py-3">
                  <dt class="text-sm text-text-muted">{{ 'profile.fields.telegram' | transloco }}</dt>
                  <dd><app-telegram-link /></dd>
                </div>
              </dl>
              <app-participant-form-card />
              <app-password-card [user]="user" />
              </div>
              <button type="button" class="ui-btn-soft mt-4 sm:hidden" (click)="logout()">
                <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-4" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
              </button>
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
                  <span class="icon-[heroicons--trophy] size-4" aria-hidden="true"></span>{{ 'profile.achievements.open' | transloco }}
                </a>
              </div>
            }
            @case ('favorites') {
              @if (favorites.error()) {
                <app-load-error (retry)="favorites.reload()" />
              } @else if (favoriteCount() > 0) {
                <ul class="motion-reveal grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
                  @for (tour of favoritesValue()?.tours; track tour.id) {
                    <li><app-tour-card [tour]="tour" /></li>
                  }
                </ul>
              } @else if (favorites.hasValue()) {
                <app-state-block icon="icon-[heroicons--heart]" title="favorites.empty" text="favorites.emptyHint">
                  <a appSiteLink="/tours" class="ui-btn-primary"><span class="icon-[heroicons--map] size-4" aria-hidden="true"></span>{{ 'booking.browseTours' | transloco }}</a>
                </app-state-block>
              } @else {
                <app-list-skeleton class="motion-delayed" [count]="3" />
              }
            }
            @case ('history') {
              <div class="flex flex-col gap-8">
                @if (upcoming().length > 0) {
                  <section aria-labelledby="history-upcoming">
                    <h2 id="history-upcoming" class="mb-3 font-semibold text-text">{{ 'profile.history.upcoming' | transloco }}</h2>
                    <ul class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      @for (tour of upcoming(); track tour.id) {
                        <li><app-tour-card [tour]="tour" /></li>
                      }
                    </ul>
                  </section>
                }
                @if (past().length > 0) {
                  <section aria-labelledby="history-past">
                    <h2 id="history-past" class="mb-3 font-semibold text-text">{{ 'profile.history.past' | transloco }}</h2>
                    <ul class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      @for (tour of past(); track tour.id) {
                        <li><app-tour-card [tour]="tour" /></li>
                      }
                    </ul>
                  </section>
                }
                @if (joined.hasValue() && upcoming().length + past().length === 0) {
                  <app-state-block icon="icon-[heroicons--flag]" title="profile.history.empty" text="profile.history.emptyHint">
                    <a appSiteLink="/tours" class="ui-btn-primary">{{ 'profile.history.cta' | transloco }}</a>
                  </app-state-block>
                }
              </div>
            }
            @case ('listings') {
              @if (auth.isPartner()) {
                <div class="flex flex-col gap-3">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <p class="text-sm text-text-muted">{{ 'cabinet.listings.subtitle' | transloco }}</p>
                    <div class="flex gap-2">
                      <a appSiteLink="/partner/listings" class="ui-btn-soft"><span class="icon-[heroicons--squares-2x2] size-4" aria-hidden="true"></span>{{ 'cabinet.open' | transloco }}</a>
                      <a appSiteLink="/tours/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'tours.addButton' | transloco }}</a>
                    </div>
                  </div>
                  @if (listingItems(); as items) {
                    @if (items.length > 0) {
                      <ul class="motion-reveal flex flex-col gap-2.5">
                        @for (item of items; track item.key) {
                          <li class="ui-card flex items-center gap-3 px-4 py-3">
                            <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-brand-500/10 text-brand-600 dark:text-brand-300">
                              <span [class]="item.icon + ' size-[1.1rem]'" aria-hidden="true"></span>
                            </span>
                            <div class="min-w-0 flex-1">
                              <p class="flex flex-wrap items-center gap-1.5">
                                <span class="ui-badge ui-badge-soft">{{ item.kind | transloco }}</span>
                                @if (item.draft) {
                                  <span class="ui-badge ui-badge-muted">{{ 'profile.listings.draft' | transloco }}</span>
                                }
                              </p>
                              <a [appSiteLink]="item.path" class="mt-0.5 block truncate font-semibold text-text hover:underline">{{ item.title }}</a>
                            </div>
                            @if (item.editPath) {
                              <a [appSiteLink]="item.editPath" class="ui-btn-soft shrink-0"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span><span class="max-sm:sr-only">{{ 'listing.edit' | transloco }}</span></a>
                            }
                          </li>
                        }
                      </ul>
                    } @else {
                      <app-state-block icon="icon-[heroicons--rectangle-stack]" title="profile.listings.empty" text="profile.listings.emptyHint">
                        <a appSiteLink="/tours/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'tours.addButton' | transloco }}</a>
                      </app-state-block>
                    }
                  } @else {
                    <app-list-skeleton class="motion-delayed" variant="rows" [count]="2" />
                  }
                </div>
              } @else {
                <app-state-block icon="icon-[heroicons--rectangle-stack]" title="profile.listings.partnersOnly" text="profile.partner.pitch">
                  <button type="button" class="ui-btn-primary" (click)="open('partner')">{{ 'profile.partner.applyCta' | transloco }}</button>
                </app-state-block>
              }
            }
            @case ('partner') {
              <div class="flex flex-col gap-4">
                @switch (user.partnerStatus) {
                  @case (statuses.APPROVED) {
                    @if (auth.isDriver()) {
                      <div class="ui-banner ui-banner-success">
                        <span class="icon-[heroicons--check-badge]" aria-hidden="true"></span>
                        <span>{{ 'profile.partner.approvedDriver' | transloco }}</span>
                      </div>
                      <div class="flex flex-wrap gap-2">
                        <a appSiteLink="/partner" class="ui-btn-primary"><span class="icon-[heroicons--squares-2x2] size-4" aria-hidden="true"></span>{{ 'cabinet.open' | transloco }}</a>
                        <a appSiteLink="/partner/fleet" class="ui-btn-soft"><span class="icon-[heroicons--truck] size-4" aria-hidden="true"></span>{{ 'cabinet.nav.fleet' | transloco }}</a>
                        <a appSiteLink="/partner/driver-profile" class="ui-btn-soft"><span class="icon-[heroicons--identification] size-4" aria-hidden="true"></span>{{ 'cabinet.nav.driverProfile' | transloco }}</a>
                      </div>
                    } @else {
                    <div class="ui-banner ui-banner-success">
                      <span class="icon-[heroicons--check-badge]" aria-hidden="true"></span>
                      <span>{{ 'profile.partner.approved' | transloco }}</span>
                    </div>
                    <div class="flex flex-wrap gap-2">
                      <a appSiteLink="/partner" class="ui-btn-primary"><span class="icon-[heroicons--squares-2x2] size-4" aria-hidden="true"></span>{{ 'cabinet.open' | transloco }}</a>
                      <a appSiteLink="/achievements/award" class="ui-btn-soft"><span class="icon-[heroicons--trophy] size-4" aria-hidden="true"></span>{{ 'profile.partner.awardCta' | transloco }}</a>
                      <a appSiteLink="/tours/new" class="ui-btn-soft"><span class="icon-[heroicons--map] size-4" aria-hidden="true"></span>{{ 'profile.partner.addTourCta' | transloco }}</a>
                      <a appSiteLink="/blog/new" class="ui-btn-soft"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'blog.form.newTitle' | transloco }}</a>
                    </div>
                    }
                  }
                  @case (statuses.PENDING) {
                    <div class="ui-banner ui-banner-info">
                      <span class="icon-[heroicons--clock]" aria-hidden="true"></span>
                      <span>{{ 'profile.partner.pending' | transloco }} {{ (user.partnerKind === 'driver' ? 'profile.partner.kind.driver' : 'profile.partner.kind.organizer') | transloco }}.</span>
                    </div>
                  }
                  @default {
                    @if (user.partnerStatus === statuses.REJECTED) {
                      <div class="ui-banner ui-banner-danger">
                        <span class="icon-[heroicons--no-symbol]" aria-hidden="true"></span>
                        <span>{{ 'profile.partner.rejected' | transloco }}</span>
                      </div>
                    }
                    <app-state-block icon="icon-[heroicons--briefcase]" title="profile.partner.pitchTitle" text="profile.partner.pitch">
                      <div class="grid w-full max-w-md gap-2 text-left sm:grid-cols-2" role="radiogroup" [attr.aria-label]="'profile.partner.kindLabel' | transloco">
                        @for (option of kinds; track option.kind) {
                          <button type="button" role="radio" class="ui-card flex flex-col gap-1 p-3 text-sm transition-colors" [class.ring-2]="kind() === option.kind" [class.ring-brand-500]="kind() === option.kind" [attr.aria-checked]="kind() === option.kind" (click)="kind.set(option.kind)">
                            <span class="flex items-center gap-2 font-semibold text-text"><span [class]="option.icon + ' size-5 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ 'profile.partner.kind.' + option.kind | transloco }}</span>
                            <span class="text-xs text-text-muted">{{ 'profile.partner.kindHint.' + option.kind | transloco }}</span>
                          </button>
                        }
                      </div>
                      <button type="button" class="ui-btn-primary" [disabled]="applying()" [attr.aria-busy]="applying()" (click)="apply()">
                        @if (applying()) {
                          <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>
                        }
                        {{ 'profile.partner.applyCta' | transloco }}
                      </button>
                    </app-state-block>
                    <p class="text-center text-xs text-text-muted">{{ 'profile.partner.publishOnly' | transloco }}</p>
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
  private readonly notify = inject(Notify);
  protected readonly statuses = PartnerStatus;
  // What kind of partner the visitor applies as.
  protected readonly kind = signal<PartnerKind>('organizer');
  protected readonly kinds = [
    { kind: 'organizer', icon: 'icon-[heroicons--map]' },
    { kind: 'driver', icon: 'icon-[heroicons--truck]' },
  ] as const;

  protected readonly active = computed(
    (): Tab => TABS.find((item) => item.key === this.tab())?.key ?? 'info',
  );
  protected readonly displayName = computed(() => {
    const user = this.auth.user();
    return user?.firstName ?? user?.username ?? user?.email ?? '';
  });

  // The tours signed up for and medals feed the summary on top, so they
  // load at once; the hall of fame loads when its tab opens.
  // Reloaded when a heart changes (here or on another page).
  private readonly favoritesStore = inject(FavoritesStore);
  protected readonly favorites = httpResource<Favorites>(() => {
    this.favoritesStore.version();
    return this.active() === 'favorites' ? '/favorites/mine' : undefined;
  });
  protected readonly favoritesValue = computed(() => loaded(this.favorites));
  protected readonly favoriteCount = computed(() => {
    const value = this.favoritesValue();
    return value?.tours.length ?? 0;
  });
  private readonly lang = injectLang();
  private readonly transloco = inject(TranslocoService);
  protected readonly joined = httpResource<Tour[]>(() => '/tours/joined');
  private readonly achievements = httpResource<MyAchievements>(() => '/achievements/me');
  private readonly hall = httpResource<HallOfFame>(() =>
    this.active() === 'achievements' ? '/hall-of-fame' : undefined,
  );
  private readonly catalog = httpResource<Achievement[]>(() =>
    this.active() === 'achievements' ? '/achievements' : undefined,
  );
  protected readonly achievementsValue = computed(() => loaded(this.achievements));
  protected readonly catalogList = computed(() => loaded(this.catalog) ?? []);
  protected readonly leaders = computed(() => loaded(this.hall)?.leaders ?? []);
  protected readonly myRank = computed(
    () => this.leaders().find((entry) => entry.user.id === this.auth.user()?.id)?.rank ?? null,
  );
  // Tours ahead, soonest first, then the ones behind, latest first.
  protected readonly upcoming = computed(() => {
    const today = todayInBishkek();
    return (loaded(this.joined) ?? [])
      .filter((tour) => tour.startDate >= today)
      .sort((a, b) => a.startDate.localeCompare(b.startDate));
  });
  protected readonly past = computed(() => {
    const today = todayInBishkek();
    return (loaded(this.joined) ?? []).filter((tour) => tour.startDate < today);
  });
  // A partner's listings, every kind in one list (the cabinet has the details).
  private readonly listingsOpen = computed(
    () => this.active() === 'listings' && this.auth.isPartner(),
  );
  private readonly myTours = httpResource<Tour[]>(() =>
    this.listingsOpen() ? '/tours/mine' : undefined,
  );
  private readonly myPosts = httpResource<BlogPost[]>(() =>
    this.listingsOpen() ? '/blog/mine' : undefined,
  );
  // null until everything has answered (a failed kind counts as empty).
  protected readonly listingItems = computed((): ListingItem[] | null => {
    const sources = [this.myTours, this.myPosts];
    if (sources.some((source) => source.isLoading())) return null;
    return [
      ...(loaded(this.myTours) ?? []).map((tour) => ({
        key: `tour-${tour.id}`,
        kind: 'cabinet.kinds.tour',
        icon: 'icon-[heroicons--map]',
        title: tour.title,
        path: `/tours/${tour.slug}`,
        editPath: `/partner/tours/${tour.id}`,
        draft: !tour.isPublished,
      })),
      ...(loaded(this.myPosts) ?? []).map((post) => ({
        key: `post-${post.id}`,
        kind: 'profile.listings.kinds.post',
        icon: 'icon-[heroicons--newspaper]',
        title: post.title,
        path: `/blog/${post.slug}`,
        editPath: `/blog/edit/${post.id}`,
        draft: !post.isPublished,
      })),
    ];
  });

  // The summary: null while its number is loading.
  protected readonly stats = computed(() => {
    const mine = loaded(this.achievements);
    const joined = this.joined.error() ? [] : loaded(this.joined);
    const lang = this.lang();
    return [
      { key: 'tours', value: joined ? formatAmount(lang, this.past().length) : null },
      { key: 'upcoming', value: joined ? formatAmount(lang, this.upcoming().length) : null },
      { key: 'awards', value: mine ? String(mine.awards.length) : null },
    ];
  });
  protected readonly applying = signal(false);
  protected readonly applyFailed = signal(false);

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('profile.title'),
      description: this.transloco.translate('login.subtitle'),
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
      await this.auth.applyAsPartner(this.kind());
      this.notify.success({ title: 'profile.partner.applied' });
    } catch {
      this.applyFailed.set(true);
      this.notify.error({ title: 'tours.signup.errors.generic', text: 'notify.tryAgain' });
    } finally {
      this.applying.set(false);
    }
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    this.notify.info({ title: 'notify.signedOut' });
    await this.navigate('/');
  }
}
