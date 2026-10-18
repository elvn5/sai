import type { BlogPost, Hotel, LocalGuide, StatsReport, Tour, Trail } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDateRange, todayInBishkek } from '@/core/lib/format';
import { contentLang } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { injectConfirm } from '@/ui/confirm-dialog';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /partner/listings: everything the partner publishes — tours (edit,
// repeat on new dates, delete), places to stay, the guide profile, routes
// and blog posts — with each listing's views over 30 days.
@Component({
  selector: 'app-partner-listings-page',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-8">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.nav.listings' | transloco }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.listings.subtitle' | transloco }}</p>
      </div>

      <section class="flex flex-col gap-3" aria-labelledby="my-tours">
        <div class="flex items-center justify-between gap-2">
          <h2 id="my-tours" class="font-semibold text-text">{{ 'cabinet.listings.tours' | transloco }}</h2>
          <a appSiteLink="/tours/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'tours.addButton' | transloco }}</a>
        </div>
        @for (group of tourGroups(); track group.key) {
          @if (group.key === 'past' && group.items.length > 0) {
            <h3 class="mt-2 text-xs font-semibold tracking-wide text-text-muted uppercase">{{ 'cabinet.listings.pastTours' | transloco }}</h3>
          }
          @for (tour of group.items; track tour.id) {
            <div class="ui-card flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <a [appSiteLink]="'/tours/' + tour.slug" [lang]="lang(tour.locale)" class="block truncate font-medium text-text hover:underline">{{ tour.title }}</a>
                <p class="text-xs text-text-muted">
                  {{ dates(tour) }} · {{ 'cabinet.listings.views30' | transloco: { count: views().get(tour.id) ?? 0 } }} · {{ 'reviews.count' | transloco: { count: tour.reviewCount } }}
                </p>
              </div>
              <div class="flex shrink-0 flex-wrap gap-2">
                <a [appSiteLink]="'/partner/tours/' + tour.id" class="ui-btn-soft"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</a>
                <a [appSiteLink]="'/tours/new?from=' + tour.id" class="ui-btn-soft"><span class="icon-[heroicons--document-duplicate] size-4" aria-hidden="true"></span>{{ 'cabinet.listings.duplicate' | transloco }}</a>
                <button type="button" class="ui-btn-ghost" [attr.aria-label]="('tours.deleteButton' | transloco) + ': ' + tour.title" [disabled]="deleting() === tour.id" (click)="deleteTour(tour)">
                  <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
                </button>
              </div>
            </div>
          }
        }
        @if (tours.hasValue() && tourList().length === 0) {
          <p class="rounded-card border border-dashed border-border px-6 py-8 text-center text-sm text-text-muted">{{ 'cabinet.listings.noTours' | transloco }}</p>
        }
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="my-hotels">
        <div class="flex items-center justify-between gap-2">
          <h2 id="my-hotels" class="font-semibold text-text">{{ 'profile.listings.hotelsHeading' | transloco }}</h2>
          <a appSiteLink="/hotels/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'hotels.addButton' | transloco }}</a>
        </div>
        @for (hotel of hotelList(); track hotel.id) {
          <div class="ui-card flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0">
              <a [appSiteLink]="'/hotels/' + hotel.slug" class="block truncate font-medium text-text hover:underline">{{ hotel.name }}</a>
              <p class="text-xs text-text-muted">
                {{ 'destinations.' + hotel.destination | transloco }} · {{ 'cabinet.listings.views30' | transloco: { count: views().get(hotel.id) ?? 0 } }} · {{ 'reviews.count' | transloco: { count: hotel.reviewCount } }}
              </p>
            </div>
            <a [appSiteLink]="'/hotels/' + hotel.slug + '/edit'" class="ui-btn-soft shrink-0"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</a>
          </div>
        } @empty {
          @if (hotels.hasValue()) {
            <p class="rounded-card border border-dashed border-border px-6 py-8 text-center text-sm text-text-muted">{{ 'profile.listings.noHotels' | transloco }}</p>
          }
        }
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="my-guide">
        <h2 id="my-guide" class="font-semibold text-text">{{ 'profile.listings.guideHeading' | transloco }}</h2>
        @if (guide(); as profile) {
          <div class="ui-card flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0">
              <a [appSiteLink]="'/local-guides/' + profile.slug" class="block truncate font-medium text-text hover:underline">{{ profile.name }}</a>
              <p class="text-xs text-text-muted">
                {{ 'cabinet.listings.views30' | transloco: { count: views().get(profile.id) ?? 0 } }} · {{ 'reviews.count' | transloco: { count: profile.reviewCount } }}
              </p>
            </div>
            <a [appSiteLink]="'/local-guides/' + profile.slug + '/edit'" class="ui-btn-soft shrink-0"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</a>
          </div>
        } @else if (guideResource.hasValue()) {
          <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-8 text-center">
            <p class="max-w-sm text-sm text-text-muted">{{ 'profile.listings.guidePitch' | transloco }}</p>
            <a appSiteLink="/local-guides/new" class="ui-btn-primary">{{ 'localGuides.addButton' | transloco }}</a>
          </div>
        }
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="my-trails">
        <div class="flex items-center justify-between gap-2">
          <h2 id="my-trails" class="font-semibold text-text">{{ 'cabinet.listings.trails' | transloco }}</h2>
          <a appSiteLink="/trails/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'cabinet.listings.addTrail' | transloco }}</a>
        </div>
        @for (trail of trailList(); track trail.id) {
          <div class="ui-card flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0">
              <a [appSiteLink]="'/trails/' + trail.slug" class="block truncate font-medium text-text hover:underline">{{ trail.name }}</a>
              @if (!trail.isPublished) {
                <p class="text-xs text-text-muted">{{ 'trails.draft' | transloco }}</p>
              }
            </div>
            <a [appSiteLink]="'/trails/' + trail.slug + '/edit'" class="ui-btn-soft shrink-0"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</a>
          </div>
        } @empty {
          @if (trails.hasValue()) {
            <p class="rounded-card border border-dashed border-border px-6 py-8 text-center text-sm text-text-muted">{{ 'cabinet.listings.noTrails' | transloco }}</p>
          }
        }
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="my-posts">
        <div class="flex items-center justify-between gap-2">
          <h2 id="my-posts" class="font-semibold text-text">{{ 'profile.listings.postsHeading' | transloco }}</h2>
          <a appSiteLink="/blog/new" class="ui-btn-primary"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'blog.form.newTitle' | transloco }}</a>
        </div>
        @for (post of postList(); track post.id) {
          <div class="ui-card flex items-center justify-between gap-4 px-4 py-3">
            <div class="min-w-0">
              @if (post.isPublished) {
                <a [appSiteLink]="'/blog/' + post.slug" [lang]="lang(post.locale)" class="block truncate font-medium text-text hover:underline">{{ post.title }}</a>
              } @else {
                <span class="block truncate font-medium text-text">{{ post.title }}</span>
              }
              <p class="text-xs text-text-muted">{{ (post.isPublished ? 'profile.listings.published' : 'profile.listings.draft') | transloco }}</p>
            </div>
            <div class="flex shrink-0 gap-2">
              <a [appSiteLink]="'/blog/edit/' + post.id" class="ui-btn-soft"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</a>
              <button type="button" class="ui-btn-ghost" [attr.aria-label]="('profile.listings.deletePost' | transloco) + ': ' + post.title" [disabled]="deleting() === post.id" (click)="deletePost(post)">
                <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
              </button>
            </div>
          </div>
        } @empty {
          @if (posts.hasValue()) {
            <p class="rounded-card border border-dashed border-border px-6 py-8 text-center text-sm text-text-muted">{{ 'profile.listings.noPosts' | transloco }}</p>
          }
        }
      </section>
    </div>
  `,
})
export class PartnerListingsPage {
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly currentLang = injectLang();

  protected readonly tours = httpResource<Tour[]>(() => '/tours/mine');
  protected readonly hotels = httpResource<Hotel[]>(() => '/hotels/mine');
  protected readonly guideResource = httpResource<{ guide: LocalGuide | null }>(
    () => '/local-guides/mine',
  );
  protected readonly trails = httpResource<Trail[]>(() => '/trails/mine');
  protected readonly posts = httpResource<BlogPost[]>(() => '/blog/mine');
  private readonly stats = httpResource<StatsReport>(() => ({
    url: '/listing-stats/mine',
    params: { days: 30 },
  }));

  protected readonly tourList = computed(() => loaded(this.tours) ?? []);
  protected readonly hotelList = computed(() => loaded(this.hotels) ?? []);
  protected readonly guide = computed(() => loaded(this.guideResource)?.guide ?? null);
  protected readonly trailList = computed(() => loaded(this.trails) ?? []);
  protected readonly postList = computed(() => loaded(this.posts) ?? []);
  // Upcoming tours soonest first; past ones most recent first.
  protected readonly tourGroups = computed(() => {
    const today = todayInBishkek();
    return [
      {
        key: 'upcoming',
        items: this.tourList()
          .filter((tour) => tour.endDate >= today)
          .sort((a, b) => a.startDate.localeCompare(b.startDate)),
      },
      { key: 'past', items: this.tourList().filter((tour) => tour.endDate < today) },
    ];
  });
  // Views over the last 30 days, by listing id.
  protected readonly views = computed(
    () => new Map((loaded(this.stats)?.listings ?? []).map((row) => [row.listing.id, row.views])),
  );
  // The id of what's being deleted.
  protected readonly deleting = signal<string | null>(null);
  protected readonly lang = contentLang;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.listings'),
      description: this.transloco.translate('cabinet.listings.subtitle'),
      robots: 'noindex',
    }));
  }

  protected dates(tour: Tour): string {
    return formatDateRange(this.currentLang(), tour.startDate, tour.endDate);
  }

  protected async deleteTour(tour: Tour): Promise<void> {
    if (
      !(await this.confirm({
        title: 'cabinet.listings.deleteTourConfirm',
        params: { title: tour.title },
        danger: true,
      }))
    )
      return;
    await this.remove(tour.id, `/tours/${tour.id}`);
    this.tours.reload();
    this.stats.reload();
  }

  protected async deletePost(post: BlogPost): Promise<void> {
    if (
      !(await this.confirm({
        title: 'profile.listings.deletePostConfirm',
        params: { title: post.title },
        danger: true,
      }))
    )
      return;
    await this.remove(post.id, `/blog/${post.id}`);
    this.posts.reload();
  }

  private async remove(id: string, url: string): Promise<void> {
    this.deleting.set(id);
    try {
      await firstValueFrom(this.http.delete(url));
    } finally {
      this.deleting.set(null);
    }
  }
}
