import { Analytics, trackItemView } from '@/core/analytics/analytics';
import type { Outing } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek, personName } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { LEVELS, TRANSPORTS, optionIcon } from '@/core/lib/trail-options';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { TelegramNotice } from '@/pages/outings/telegram-notice';
import { injectConfirm } from '@/ui/confirm-dialog';
import { LoadError } from '@/ui/load-error';
import { PersonAvatar } from '@/ui/person-avatar';
import { RatingInput } from '@/ui/rating-input';
import { SiteLink } from '@/ui/site-link';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  PLATFORM_ID,
  computed,
  effect,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { FormField, form, maxLength } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';

const WHEN: Intl.DateTimeFormatOptions = {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  hour: '2-digit',
  minute: '2-digit',
};

// One outing: when and where, who's going, and joining. The server renders
// the anonymous view; the visitor's own (joined, organizer, the contact)
// is read again in the browser once the session is known. Not indexed:
// outings are gone in days.
@Component({
  selector: 'app-outing-page',
  imports: [
    FormField,
    LoadError,
    NotFoundPage,
    PersonAvatar,
    RatingInput,
    SiteLink,
    TelegramNotice,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (outing.error()) {
      <div class="mx-auto max-w-3xl px-4 py-12"><app-load-error (retry)="outing.reload()" /></div>
    } @else if (item(); as o) {
      <section class="mx-auto max-w-3xl px-4 py-12">
        <a appSiteLink="/outings" class="text-sm text-text-muted hover:text-text">&larr; {{ 'outings.title' | transloco }}</a>

        <article class="mt-4 overflow-hidden rounded-card border border-border bg-surface">
          <header class="bg-linear-to-br from-brand-700 to-secondary-800 px-6 py-6 text-white">
            @if (o.status === 'cancelled') {
              <span class="mb-2 inline-block rounded-full bg-black/30 px-2.5 py-1 text-xs font-semibold">{{ 'outings.cancelled' | transloco }}</span>
            }
            <h1 class="text-2xl font-black sm:text-3xl">{{ o.title }}</h1>
            <p class="mt-2 flex items-center gap-2 text-white/90">
              <span class="icon-[heroicons--calendar-days] size-5" aria-hidden="true"></span>{{ when(o.startsAt) }}
            </p>
            @if (o.returnsAt) {
              <p class="mt-1 flex items-center gap-2 text-sm text-white/80">
                <span class="icon-[heroicons--home] size-5" aria-hidden="true"></span>{{ 'outings.returns' | transloco: { when: when(o.returnsAt) } }}
              </p>
            }
          </header>

          <dl class="grid gap-4 p-6 sm:grid-cols-2">
            <div class="relative pl-8">
              <dt class="text-xs text-text-muted"><span class="absolute top-0.5 left-0 icon-[heroicons--map-pin] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'outings.form.meetingPoint' | transloco }}</dt>
              <dd class="font-medium text-text">{{ o.meetingPoint }}</dd>
            </div>
            @if (o.trail; as trail) {
              <div class="relative pl-8">
                <dt class="text-xs text-text-muted"><span class="absolute top-0.5 left-0 icon-[heroicons--map] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'outings.form.trail' | transloco }}</dt>
                <dd class="font-medium text-text">{{ trail.name }}</dd>
              </div>
            }
            <div class="relative pl-8">
              <dt class="text-xs text-text-muted"><span [class]="transportIcon() + ' absolute top-0.5 left-0 size-5 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ 'outings.form.transport' | transloco }}</dt>
              <dd class="font-medium text-text">{{ 'outings.transport.' + o.transport | transloco }}</dd>
            </div>
            <div class="relative pl-8">
              <dt class="text-xs text-text-muted"><span [class]="levelIcon() + ' absolute top-0.5 left-0 size-5 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ 'outings.form.level' | transloco }}</dt>
              <dd class="font-medium text-text">{{ 'outings.level.' + o.level | transloco }}</dd>
            </div>
            @if (o.costPerPerson !== null) {
              <div class="relative pl-8">
                <dt class="text-xs text-text-muted"><span class="absolute top-0.5 left-0 icon-[heroicons--banknotes] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'outings.form.cost' | transloco }}</dt>
                <dd class="font-medium text-text">
                  @if (o.costPerPerson === 0) { {{ 'outings.free' | transloco }} } @else { {{ cost() }} {{ 'listing.currency' | transloco }} }
                </dd>
              </div>
            }
          </dl>

          @if (o.description) {
            <p class="border-t border-border px-6 py-5 whitespace-pre-line text-text">{{ o.description }}</p>
          }

          <section class="border-t border-border px-6 py-5" aria-labelledby="participants">
            <div class="mb-2 flex items-center justify-between text-sm">
              <h2 id="participants" class="font-semibold text-text">{{ 'outings.participants' | transloco }}</h2>
              <span class="text-text-muted">{{ 'outings.seatsTaken' | transloco: { taken: o.taken, seats: o.seats } }}</span>
            </div>
            <div class="mb-4 h-2 overflow-hidden rounded-full bg-text/10" role="progressbar" [attr.aria-valuenow]="o.taken" aria-valuemin="0" [attr.aria-valuemax]="o.seats" [attr.aria-label]="'outings.participants' | transloco">
              <div class="h-full rounded-full bg-brand-600 transition-all duration-500" [style.width.%]="filled()"></div>
            </div>
            <ul class="flex flex-wrap items-center gap-3">
              <li>
                <a [appSiteLink]="'/organizers/' + o.organizer.id" class="flex items-center gap-2 rounded-full bg-text/5 py-1 pr-3 pl-1 hover:bg-brand-500/10" [attr.title]="'organizer.profileLink' | transloco">
                  <app-person-avatar [person]="o.organizer" [size]="28" />
                  <span class="text-sm text-text">{{ name(o.organizer) }}</span>
                  <span class="text-xs text-brand-700 dark:text-brand-300">{{ 'outings.organizer' | transloco }}</span>
                </a>
              </li>
              @for (person of o.participants ?? []; track person.id) {
                <li class="flex items-center gap-2 rounded-full bg-text/5 py-1 pr-3 pl-1">
                  <app-person-avatar [person]="person" [size]="28" />
                  <span class="text-sm text-text">{{ name(person) }}</span>
                </li>
              }
            </ul>
          </section>

          @if (o.contact) {
            <div class="flex items-center gap-3 border-t border-border bg-emerald-500/10 px-6 py-4">
              <span class="icon-[heroicons--chat-bubble-bottom-center-text] size-5 text-emerald-700 dark:text-emerald-300" aria-hidden="true"></span>
              <div>
                <p class="text-xs text-text-muted">{{ 'outings.contact' | transloco }}</p>
                <p class="font-semibold text-text">{{ o.contact }}</p>
              </div>
            </div>
          }

          @if (!closed() || actionError()) {
          <div class="flex flex-wrap items-center gap-3 border-t border-border px-6 py-5">
            @if (o.isOrganizer) {
              @if (!closed()) {
                <a [appSiteLink]="'/outings/' + o.id + '/edit'" class="ui-btn-soft">
                  <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
                </a>
                <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="cancel()">{{ 'outings.cancel' | transloco }}</button>
              }
            } @else if (!closed()) {
              @if (o.isJoined) {
                <button type="button" class="ui-btn-soft" [disabled]="busy()" (click)="leave()">{{ 'outings.leave' | transloco }}</button>
              } @else {
                <button type="button" class="ui-btn-primary px-4 py-2 text-base" [disabled]="left() === 0 || busy()" (click)="join()">
                  <span class="icon-[heroicons--hand-raised] size-5" aria-hidden="true"></span>{{ (left() === 0 ? 'outings.full' : 'outings.join') | transloco }}
                </button>
                @if (left() > 0) {
                  <span class="text-sm text-text-muted">{{ 'outings.joinHint' | transloco }}</span>
                }
              }
            }
            @if (actionError(); as key) {
              <p class="ui-error w-full" role="alert">{{ key | transloco }}</p>
            }
          </div>
          }

          @if (o.canRate || o.myRating) {
            <form class="flex flex-col gap-3 border-t border-border bg-amber-500/5 px-6 py-5" novalidate (submit)="rate($event)">
              <div>
                <h2 class="font-semibold text-text">{{ 'outings.rate.title' | transloco }}</h2>
                <p class="text-sm text-text-muted">{{ 'outings.rate.hint' | transloco }}</p>
              </div>
              <app-rating-input [label]="'outings.rate.yours' | transloco" [formField]="rateForm.rating" />
              <textarea class="ui-input" rows="3" [placeholder]="'outings.rate.textPlaceholder' | transloco" [formField]="rateForm.text"></textarea>
              <div class="flex flex-wrap items-center gap-3">
                <button type="submit" class="ui-btn-primary" [disabled]="busy() || rateModel().rating === 0 || !o.canRate">{{ 'outings.rate.submit' | transloco }}</button>
                @if (rateSaved()) {
                  <span class="text-sm text-emerald-700 dark:text-emerald-300" role="status">{{ 'outings.rate.saved' | transloco }}</span>
                }
                @if (rateError()) {
                  <span class="ui-error" role="alert">{{ 'outings.rate.error' | transloco }}</span>
                }
              </div>
            </form>
          }

          @if (!closed() && (o.isJoined || o.isOrganizer)) {
            <div class="border-t border-border px-6 py-4"><app-telegram-notice /></div>
          }
        </article>
      </section>
    }
  `,
})
export class OutingPage {
  readonly id = input.required<string>();

  private readonly auth = inject(AuthStore);
  private readonly analytics = inject(Analytics);
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  // The anonymous view (server-rendered).
  protected readonly outing = httpResource<Outing>(
    () => `/outings/${encodeURIComponent(this.id())}`,
  );
  // The signed-in visitor's view; reset when the outing changes.
  private readonly personal = linkedSignal<string, Outing | null>({
    source: this.id,
    computation: () => null,
  });
  protected readonly item = computed(() => this.personal() ?? loaded(this.outing));
  protected readonly notFound = computed(() => {
    const error = this.outing.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });

  protected readonly left = computed(() => {
    const o = this.item();
    return o ? Math.max(0, o.seats - o.taken) : 0;
  });
  protected readonly filled = computed(() => {
    const o = this.item();
    return o && o.seats > 0 ? Math.min(100, (o.taken / o.seats) * 100) : 0;
  });
  protected readonly started = computed(() => {
    const o = this.item();
    return Boolean(o && new Date(o.startsAt).getTime() <= Date.now());
  });
  protected readonly closed = computed(() => this.item()?.status === 'cancelled' || this.started());
  protected readonly cost = computed(() =>
    formatAmount(this.lang(), this.item()?.costPerPerson ?? 0),
  );
  protected readonly transportIcon = computed(() =>
    optionIcon(TRANSPORTS, this.item()?.transport ?? ''),
  );
  protected readonly levelIcon = computed(() => optionIcon(LEVELS, this.item()?.level ?? ''));
  protected readonly busy = signal(false);
  // Translation key of the last failed action.
  protected readonly actionError = signal<string | null>(null);

  // The participant's rating, filled from their own once it is known.
  protected readonly rateModel = linkedSignal(() => {
    const mine = this.personal()?.myRating;
    return { rating: mine?.rating ?? 0, text: mine?.text ?? '' };
  });
  protected readonly rateForm = form(this.rateModel, (path) => {
    maxLength(path.text, 1000);
  });
  protected readonly rateSaved = signal(false);
  protected readonly rateError = signal(false);

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const o = this.item();
      if (!o) return undefined;
      const heading = this.transloco.translate('outings.title');
      return {
        title: o.title,
        description: `${this.when(o.startsAt)} · ${o.meetingPoint}`,
        robots: 'noindex',
        jsonLd: [
          seo.breadcrumbs([
            { name: heading, path: '/outings' },
            { name: o.title, path: `/outings/${o.id}` },
          ]),
        ],
      };
    });

    trackItemView(
      'view_outing',
      computed(() => this.item()?.id),
    );

    // Signed in (now or later): read the visitor's own view.
    if (isPlatformBrowser(inject(PLATFORM_ID))) {
      void this.auth.ensureSession();
      effect(() => {
        const id = this.id();
        if (this.auth.user()) void this.loadPersonal(id);
      });
    }
  }

  protected when(iso: string): string {
    return formatInBishkek(this.lang(), iso, WHEN);
  }

  protected name(person: Outing['organizer']): string {
    return personName(person);
  }

  protected async join(): Promise<void> {
    if (!(await this.auth.ensureSession())) {
      await this.navigate(`/login?redirect=${encodeURIComponent(this.router.url)}`);
      return;
    }
    const id = this.id();
    if (await this.act(this.http.post<Outing>(`/outings/${id}/join`, {}))) {
      this.analytics.track('join_outing', { outing_id: id });
    }
  }

  protected async leave(): Promise<void> {
    const id = this.id();
    if (await this.act(this.http.delete<Outing>(`/outings/${id}/join`))) {
      this.analytics.track('leave_outing', { outing_id: id });
    }
  }

  protected async cancel(): Promise<void> {
    if (!(await this.confirm({ title: 'outings.cancelConfirm', danger: true }))) return;
    await this.act(this.http.post<Outing>(`/outings/${this.id()}/cancel`, {}));
  }

  protected async rate(event: Event): Promise<void> {
    event.preventDefault();
    const { rating, text } = this.rateModel();
    if (rating === 0) return;
    this.busy.set(true);
    this.rateSaved.set(false);
    this.rateError.set(false);
    try {
      const outing = await firstValueFrom(
        this.http.put<Outing>(`/outings/${this.id()}/rating`, {
          rating,
          text: text.trim() || null,
        }),
      );
      this.personal.set(outing);
      this.rateSaved.set(true);
    } catch {
      this.rateError.set(true);
    } finally {
      this.busy.set(false);
    }
  }

  private async loadPersonal(id: string): Promise<void> {
    try {
      // Not from the transfer cache: that holds the server's anonymous answer.
      const outing = await firstValueFrom(
        this.http.get<Outing>(`/outings/${encodeURIComponent(id)}`, { transferCache: false }),
      );
      if (id === this.id()) this.personal.set(outing);
    } catch {
      // The anonymous view stays.
    }
  }

  // Join, leave or cancel: the API answers with the updated outing.
  // Whether the API took it.
  private async act(request: Observable<Outing>): Promise<boolean> {
    this.busy.set(true);
    this.actionError.set(null);
    try {
      this.personal.set(await firstValueFrom(request));
      return true;
    } catch (error) {
      this.actionError.set(
        error instanceof HttpErrorResponse && error.status === 409
          ? 'outings.errors.conflict'
          : 'outings.errors.generic',
      );
      void this.loadPersonal(this.id());
      return false;
    } finally {
      this.busy.set(false);
    }
  }
}
