import type { Achievement, HallEntry, HallOfFame, MyAchievements } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { personName } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { AwardReveal } from '@/pages/hall-of-fame/award-reveal';
import { RecentAwardsTicker } from '@/pages/hall-of-fame/recent-awards-ticker';
import { AchievementMedal } from '@/ui/achievements/achievement-medal';
import { AchievementShelf } from '@/ui/achievements/achievement-shelf';
import { AwardCodeCard } from '@/ui/achievements/award-code-card';
import { HallPodium } from '@/ui/achievements/hall-podium';
import { LoadError } from '@/ui/load-error';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// Travelers ranked by the achievements partners awarded them. The
// leaderboard renders on the server; the visitor's own shelf, award code and
// the reveal of new awards come after the session check in the browser.
@Component({
  selector: 'app-hall-of-fame-page',
  imports: [
    AchievementMedal,
    AchievementShelf,
    AwardCodeCard,
    AwardReveal,
    HallPodium,
    LoadError,
    PersonAvatar,
    RecentAwardsTicker,
    SiteLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .hero-bg {
      background:
        radial-gradient(ellipse at 50% 0%, rgb(251 191 36 / 0.25), transparent 60%),
        linear-gradient(160deg, var(--brand-900), #0b1020 70%);
    }
    .particle {
      width: calc(3px + (var(--n) * 7 % 5) * 1px);
      height: calc(3px + (var(--n) * 7 % 5) * 1px);
      left: calc(var(--n) * 7%);
      bottom: -10px;
      background: rgb(253 230 138 / 0.8);
      box-shadow: 0 0 10px 2px rgb(251 191 36 / 0.5);
      animation: float calc(6s + var(--n) * 0.5s) linear infinite;
      animation-delay: calc(var(--n) * -0.8s);
    }
    .trophy { animation: hover 3.5s ease-in-out infinite; }
    .rays {
      background: repeating-conic-gradient(from 0deg, rgb(251 191 36 / 0.22) 0deg 8deg, transparent 8deg 18deg);
      mask-image: radial-gradient(circle, #000 15%, transparent 65%);
      animation: rotate 20s linear infinite;
    }
    .guest-card { min-height: 282px; }
    @media (min-width: 640px) { .guest-card { min-height: 258px; } }
    .enter { animation: enter 0.6s cubic-bezier(0.22, 1, 0.36, 1) both; animation-delay: calc(var(--i) * 60ms); }
    @keyframes float { from { transform: translateY(0); opacity: 0; } 15% { opacity: 1; } to { transform: translateY(-520px); opacity: 0; } }
    @keyframes hover { 50% { transform: translateY(-10px); } }
    @keyframes rotate { to { transform: rotate(360deg); } }
    @keyframes enter { from { opacity: 0; transform: translateY(16px); } }
    @media (prefers-reduced-motion: reduce) { .particle, .trophy, .rays, .enter { animation: none; } }
  `,
  template: `
    <section class="relative isolate overflow-hidden px-4 pt-20 pb-16 text-center">
      <div class="hero-bg absolute inset-0 -z-10"></div>
      @for (n of particles; track n) {
        <span class="particle absolute -z-10 rounded-full" [style.--n]="n" aria-hidden="true"></span>
      }
      <div class="trophy relative mx-auto mb-6 flex size-28 items-center justify-center" aria-hidden="true">
        <span class="rays absolute inset-[-60%]"></span>
        <span class="icon-[mdi--trophy] relative size-24 text-amber-300 drop-shadow-[0_0_25px_rgba(251,191,36,0.6)]"></span>
      </div>
      <h1 class="text-4xl font-black tracking-tight text-white sm:text-5xl">{{ 'hallOfFame.title' | transloco }}</h1>
      <p class="mx-auto mt-3 max-w-xl text-white/85">{{ 'hallOfFame.subtitle' | transloco }}</p>
    </section>

    @if (hall.error()) {
      <div class="mx-auto max-w-5xl px-4 pt-12"><app-load-error (retry)="hall.reload()" /></div>
    } @else {
      @if (recent().length > 0) {
        <div class="mx-auto -mt-6 max-w-5xl px-4">
          <app-recent-awards-ticker [awards]="recent()" />
        </div>
      }

      @if (myAchievements(); as me) {
        <section class="mx-auto max-w-5xl px-4 pt-12" aria-labelledby="mine-heading">
          <div class="flex flex-wrap items-end justify-between gap-4">
            <h2 id="mine-heading" class="text-2xl font-extrabold text-text">{{ 'hallOfFame.mine.heading' | transloco }}</h2>
            <dl class="flex gap-3">
              <div class="ui-card flex flex-col-reverse px-4 py-2 text-center">
                <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.points' | transloco }}</dt>
                <dd class="text-2xl font-black text-amber-600 dark:text-amber-400">{{ me.points }}</dd>
              </div>
              <div class="ui-card flex flex-col-reverse px-4 py-2 text-center">
                <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.earned' | transloco }}</dt>
                <dd class="text-2xl font-black text-text">{{ me.awards.length }}</dd>
              </div>
              @if (myRank(); as rank) {
                <div class="ui-card flex flex-col-reverse px-4 py-2 text-center">
                  <dt class="text-xs text-text-muted">{{ 'hallOfFame.mine.rank' | transloco }}</dt>
                  <dd class="text-2xl font-black text-brand-700 dark:text-brand-300">#{{ rank }}</dd>
                </div>
              }
            </dl>
          </div>

          <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_340px]">
            <div class="ui-card p-5">
              <app-achievement-shelf [mine]="me" [catalog]="catalog.value() ?? []" />
            </div>
            <div class="flex flex-col gap-3">
              <app-award-code-card [code]="me.code" />
              <p class="text-sm text-text-muted">{{ 'hallOfFame.mine.howTo' | transloco }}</p>
            </div>
          </div>
        </section>
      } @else if (auth.checked() && !auth.user()) {
        <section class="mx-auto max-w-5xl px-4 pt-12">
          <div class="guest-card flex flex-col items-center justify-center gap-4 rounded-card border border-dashed border-border bg-surface px-6 py-10 text-center">
            <div class="flex -space-x-3" aria-hidden="true">
              @for (achievement of catalogPreview(); track achievement.id) {
                <app-achievement-medal [icon]="achievement.icon" [rarity]="achievement.rarity" />
              }
            </div>
            <p class="max-w-md text-text">{{ 'hallOfFame.guest.pitch' | transloco }}</p>
            <a appSiteLink="/login" class="ui-btn-primary">{{ 'hallOfFame.guest.cta' | transloco }}</a>
          </div>
        </section>
      } @else if (!auth.checked() || auth.user()) {
        <!-- Holds the guest block's place until the session is known (no layout shift). -->
        <section class="mx-auto max-w-5xl px-4 pt-12" aria-hidden="true">
          <div class="guest-card rounded-card border border-dashed border-border bg-surface"></div>
        </section>
      }

      <section class="mx-auto max-w-5xl px-4 py-14" aria-labelledby="leaders-heading">
        <h2 id="leaders-heading" class="mb-10 text-center text-2xl font-extrabold text-text">{{ 'hallOfFame.leaders.heading' | transloco }}</h2>
        @if (leaders().length > 0) {
          <app-hall-podium [leaders]="leaders()" />
          <ol class="mt-10 flex flex-col gap-2" [start]="4">
            @for (entry of leaders().slice(3); track entry.user.id; let i = $index) {
              <li
                class="enter ui-card flex items-center gap-4 px-4 py-3"
                [class.ring-2]="entry.user.id === auth.user()?.id"
                [class.ring-brand-500]="entry.user.id === auth.user()?.id"
                [style.--i]="i"
              >
                <span class="w-8 text-center text-lg font-black text-text-muted">{{ entry.rank }}</span>
                <app-person-avatar [person]="entry.user" />
                <div class="min-w-0 flex-1">
                  <p class="truncate font-semibold text-text">{{ name(entry.user) }}</p>
                  <div class="mt-1 flex gap-1" aria-hidden="true">
                    @for (achievement of entry.achievements.slice(0, 6); track achievement.id) {
                      <app-achievement-medal [icon]="achievement.icon" [rarity]="achievement.rarity" size="sm" />
                    }
                  </div>
                </div>
                <span class="text-right text-sm font-bold text-amber-700 dark:text-amber-400">
                  {{ 'hallOfFame.points' | transloco: { count: entry.points } }}
                </span>
              </li>
            }
          </ol>
        } @else if (hall.hasValue()) {
          <p class="text-center text-text-muted">{{ 'hallOfFame.leaders.empty' | transloco }}</p>
        }
      </section>
    }

    @if (freshAwards().length > 0 && !revealDismissed()) {
      <app-award-reveal [awards]="freshAwards()" (done)="finishReveal()" />
    }
  `,
})
export class HallOfFamePage {
  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  protected readonly particles = Array.from({ length: 14 }, (_, i) => i + 1);

  protected readonly hall = httpResource<HallOfFame>(() => '/hall-of-fame');
  protected readonly catalog = httpResource<Achievement[]>(() => '/achievements');
  // The visitor's own achievements: only once signed in (never on the server).
  protected readonly mine = httpResource<MyAchievements>(() =>
    this.auth.user() ? '/achievements/me' : undefined,
  );

  protected readonly myAchievements = computed(() => loaded(this.mine));
  protected readonly leaders = computed(() => loaded(this.hall)?.leaders ?? []);
  protected readonly recent = computed(() => loaded(this.hall)?.recent ?? []);
  protected readonly catalogPreview = computed(() => (loaded(this.catalog) ?? []).slice(0, 5));
  protected readonly myRank = computed(
    () =>
      this.leaders().find((entry: HallEntry) => entry.user.id === this.auth.user()?.id)?.rank ??
      null,
  );
  protected readonly freshAwards = computed(
    () => loaded(this.mine)?.awards.filter((award) => award.isNew) ?? [],
  );
  protected readonly revealDismissed = signal(false);

  constructor() {
    const transloco = inject(TranslocoService);
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const title = transloco.translate('hallOfFame.title');
      return {
        title,
        description: transloco.translate('hallOfFame.subtitle'),
        jsonLd: [seo.breadcrumbs([{ name: title, path: '/hall-of-fame' }])],
      };
    });
  }

  protected name(person: Parameters<typeof personName>[0]): string {
    return personName(person);
  }

  protected async finishReveal(): Promise<void> {
    this.revealDismissed.set(true);
    try {
      await firstValueFrom(this.http.post('/achievements/me/seen', {}));
      this.mine.reload();
    } catch {
      // Shown again next time: harmless.
    }
  }
}
