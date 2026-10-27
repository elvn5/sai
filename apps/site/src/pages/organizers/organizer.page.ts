import type { OrganizerProfile, OrganizerTours, Person, ReviewReply } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, personName } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { TourCard } from '@/ui/listing/tour-card';
import { LoadError } from '@/ui/load-error';
import { PersonAvatar } from '@/ui/person-avatar';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// A review of a tour, as the page lists them.
interface OrganizerReview {
  rating: number;
  text: string | null;
  createdAt: string;
  author: Person;
  // The tour it is about: its title and link.
  title: string;
  link: string;
  reply: ReviewReply | null;
}

// An organizer's public page: who they are, how they are rated by those
// who went with them, what they ran and what is coming up. Verified means
// an approved partner.
@Component({
  selector: 'app-organizer-page',
  imports: [LoadError, NotFoundPage, PersonAvatar, RatingStars, SiteLink, TourCard, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (failed()) {
      <div class="mx-auto max-w-5xl px-4 py-12"><app-load-error (retry)="reload()" /></div>
    } @else if (profile(); as p) {
      <section class="mx-auto max-w-5xl px-4 py-10">
        <header class="ui-card flex flex-col gap-5 p-6 sm:flex-row sm:items-center">
          <app-person-avatar [person]="p" [size]="88" class="ring-4 ring-brand-500/15" />
          <div class="min-w-0 flex-1">
            <p class="text-xs font-semibold tracking-wide text-brand-700 uppercase dark:text-brand-300">{{ 'organizer.kicker' | transloco }}</p>
            <div class="mt-1 flex flex-wrap items-center gap-2">
              <h1 class="text-2xl font-black text-text sm:text-3xl">{{ name() }}</h1>
              @if (p.isVerified) {
                <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/12 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                  <span class="icon-[heroicons--check-badge-solid] size-4" aria-hidden="true"></span>{{ 'organizer.verified' | transloco }}
                </span>
              }
            </div>
            <p class="mt-1 text-sm text-text-muted">
              @if (p.username) {
                <span>&#64;{{ p.username }} · </span>
              }
              {{ 'organizer.since' | transloco: { year: sinceYear() } }}
            </p>
            @if (p.isVerified) {
              <p class="mt-2 text-sm text-text-muted">{{ 'organizer.verifiedHint' | transloco }}</p>
            }
          </div>
          <div class="flex shrink-0 flex-col items-start gap-1 sm:items-end">
            @if (rating().count > 0) {
              <span class="text-4xl leading-none font-black text-text">{{ rating().average.toFixed(1) }}</span>
              <app-rating-stars [value]="rating().average" size="md" />
              <span class="text-sm text-text-muted">{{ 'organizer.reviewsCount' | transloco: { count: rating().count } }}</span>
            } @else {
              <span class="text-sm text-text-muted">{{ 'organizer.noRating' | transloco }}</span>
            }
          </div>
        </header>

        <dl class="mt-4 grid grid-cols-3 gap-3">
          @for (stat of stats(); track stat.label) {
            <div class="ui-card p-4">
              <dt class="text-xs text-text-muted">{{ stat.label | transloco }}</dt>
              <dd class="mt-1 text-2xl font-black text-text">{{ stat.value }}</dd>
            </div>
          }
        </dl>

        @if (tours()?.upcoming?.length) {
          <section class="mt-10" aria-labelledby="org-tours">
            <h2 id="org-tours" class="mb-4 text-xl font-bold text-text">{{ 'organizer.upcomingTours' | transloco }}</h2>
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              @for (t of tours()!.upcoming; track t.id) {
                <app-tour-card [tour]="t" />
              }
            </div>
          </section>
        }

        <section class="mt-10" aria-labelledby="org-reviews">
          <h2 id="org-reviews" class="mb-4 text-xl font-bold text-text">{{ 'organizer.reviews' | transloco }}</h2>
          @if (reviews().length === 0) {
            <p class="ui-card p-6 text-text-muted">{{ 'organizer.noReviews' | transloco }}</p>
          } @else {
            <ul class="flex flex-col gap-3">
              @for (r of reviews(); track $index) {
                <li class="ui-card p-4">
                  <div class="flex items-center gap-3">
                    <app-person-avatar [person]="r.author" [size]="36" />
                    <div class="min-w-0 flex-1">
                      <p class="truncate font-semibold text-text">{{ authorName(r.author) }}</p>
                      <p class="text-xs text-text-muted">{{ date(r.createdAt) }}</p>
                    </div>
                    <app-rating-stars [value]="r.rating" />
                  </div>
                  @if (r.text) {
                    <p class="mt-3 whitespace-pre-line text-text">{{ r.text }}</p>
                  }
                  <p class="mt-2 text-xs text-text-muted">
                    {{ 'organizer.aboutTour' | transloco }} <a [appSiteLink]="r.link" class="font-medium text-brand-700 hover:underline dark:text-brand-300">{{ r.title }}</a>
                  </p>
                  @if (r.reply; as reply) {
                    <div class="mt-3 rounded-lg bg-text/5 p-3 text-sm">
                      <p class="text-xs font-semibold text-text-muted">{{ 'organizer.reply' | transloco }}</p>
                      <p class="mt-1 whitespace-pre-line text-text">{{ reply.text }}</p>
                    </div>
                  }
                </li>
              }
            </ul>
          }
        </section>
      </section>
    }
  `,
})
export class OrganizerPage {
  readonly id = input.required<string>();

  private readonly lang = injectLang();
  private readonly transloco = inject(TranslocoService);

  private readonly profileRes = httpResource<OrganizerProfile>(
    () => `/users/${encodeURIComponent(this.id())}/public`,
  );
  private readonly toursRes = httpResource<OrganizerTours>(() => ({
    url: `/tours/organizers/${encodeURIComponent(this.id())}`,
    params: { locale: this.lang() },
  }));

  protected readonly profile = computed(() => loaded(this.profileRes));
  protected readonly tours = computed(() => loaded(this.toursRes));

  protected readonly notFound = computed(() => {
    const error = this.profileRes.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly failed = computed(
    () => !this.notFound() && Boolean(this.profileRes.error() || this.toursRes.error()),
  );

  protected readonly name = computed(() => {
    const p = this.profile();
    return p ? personName(p) : '';
  });
  protected readonly sinceYear = computed(() => {
    const p = this.profile();
    return p ? new Date(p.memberSince).getFullYear() : '';
  });

  protected readonly rating = computed(() => this.tours()?.rating ?? { average: 0, count: 0 });

  protected readonly stats = computed(() => {
    const t = this.tours();
    return [
      { label: 'organizer.stats.tours', value: t?.held ?? 0 },
      { label: 'organizer.stats.people', value: t?.participants ?? 0 },
      { label: 'organizer.stats.reviews', value: this.rating().count },
    ];
  });

  protected readonly reviews = computed((): OrganizerReview[] =>
    (this.tours()?.reviews ?? []).map(
      (r): OrganizerReview => ({
        rating: r.rating,
        text: r.text,
        createdAt: r.createdAt,
        author: r.author,
        title: r.listing.title,
        link: `/tours/${r.listing.slug}`,
        reply: r.reply,
      }),
    ),
  );

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const p = this.profile();
      if (!p || !this.tours()) return undefined;
      const name = personName(p);
      const { average, count } = this.rating();
      const description =
        count > 0
          ? this.transloco.translate('organizer.metaRated', {
              name,
              rating: average.toFixed(1),
              count,
            })
          : this.transloco.translate('organizer.meta', { name });
      const path = `/organizers/${p.id}`;
      return {
        title: this.transloco.translate('organizer.title', { name }),
        description,
        type: 'profile',
        image: p.avatarUrl,
        jsonLd: [
          {
            '@context': 'https://schema.org',
            '@type': 'ProfilePage',
            url: seo.pageUrl(path),
            dateCreated: p.memberSince,
            mainEntity: {
              '@type': 'Person',
              name,
              ...(p.username ? { alternateName: `@${p.username}` } : {}),
            },
          },
          seo.breadcrumbs([{ name, path }]),
        ],
      };
    });
  }

  protected reload(): void {
    this.profileRes.reload();
    this.toursRes.reload();
  }

  protected authorName(person: Person): string {
    return person.firstName || person.username
      ? personName(person)
      : this.transloco.translate('reviews.anonymous');
  }

  protected date(iso: string): string {
    return formatInBishkek(this.lang(), iso, { day: 'numeric', month: 'long', year: 'numeric' });
  }
}
