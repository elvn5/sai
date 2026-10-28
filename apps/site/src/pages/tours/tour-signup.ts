import { Analytics } from '@/core/analytics/analytics';
import type { Tour, TourParticipant } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { openChat } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, personName } from '@/core/lib/format';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { TelegramNotice } from '@/pages/tours/telegram-notice';
import { injectConfirm } from '@/ui/confirm-dialog';
import { PersonAvatar } from '@/ui/person-avatar';
import { RatingInput } from '@/ui/rating-input';
import { SiteLink } from '@/ui/site-link';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  PLATFORM_ID,
  computed,
  effect,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, maxLength } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';

const WHEN: Intl.DateTimeFormatOptions = {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  hour: '2-digit',
  minute: '2-digit',
};

// Used when a tour has no meeting time, as the API's reminders assume.
const DEFAULT_MEETING_TIME = '09:00';

// Signing up for a tour: where and when the group meets, places taken,
// and the visitor's own place. The organizer sees who signed up and who
// came; a participant rates the tour after it (their review of it). The
// server renders the anonymous view; the visitor's own is read in the
// browser once the session is known.
@Component({
  selector: 'app-tour-signup',
  imports: [FormField, PersonAvatar, RatingInput, SiteLink, TelegramNotice, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (item(); as t) {
      <section class="ui-card flex flex-col gap-4 p-4" aria-labelledby="signup-heading">
        <div>
          <h2 id="signup-heading" class="font-bold text-text">{{ 'tours.signup.heading' | transloco }}</h2>
          <p class="mt-1 flex items-center gap-2 text-sm text-text">
            <span class="icon-[heroicons--clock] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'tours.signup.meeting' | transloco: { when: when() } }}
          </p>
          @if (t.meetingPoint) {
            <p class="mt-1 flex items-center gap-2 text-sm text-text">
              <span class="icon-[heroicons--map-pin] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ t.meetingPoint }}
            </p>
          }
        </div>

        <div>
          <p class="text-sm text-text-muted">
            @if (t.capacity !== null) {
              {{ 'tours.signup.placesTaken' | transloco: { taken: t.taken, capacity: t.capacity } }}
            } @else {
              {{ 'tours.signup.signedUp' | transloco: { taken: t.taken } }}
            }
          </p>
          @if (t.capacity !== null) {
            <div class="mt-2 h-2 overflow-hidden rounded-full bg-text/10" role="progressbar" [attr.aria-valuenow]="t.taken" aria-valuemin="0" [attr.aria-valuemax]="t.capacity" [attr.aria-label]="'tours.signup.heading' | transloco">
              <div class="h-full rounded-full bg-brand-600 transition-all duration-500" [style.width.%]="filled()"></div>
            </div>
          }
        </div>

        @if (t.isOrganizer) {
          <div>
            <div class="mb-2 flex items-center justify-between gap-2">
              <h3 class="text-sm font-semibold text-text">{{ 'tours.signup.participants' | transloco }}</h3>
              <a [appSiteLink]="'/partner/tours/' + t.id + '/participants'" class="ui-btn-soft ui-btn-xs"><span class="icon-[heroicons--user-group] size-4" aria-hidden="true"></span>{{ 'tours.signup.manageGroup' | transloco }}</a>
            </div>
            @if (t.waitlist) {
              <p class="mb-2 text-sm text-text-muted">{{ 'tours.signup.waitlistCount' | transloco: { count: t.waitlist } }}</p>
            }
            @if (t.roster?.length) {
              <ul class="flex flex-col gap-2">
                @for (person of t.roster; track person.id) {
                  <li class="flex items-center gap-2 text-sm">
                    <app-person-avatar [person]="person" [size]="28" />
                    <span class="min-w-0 flex-1 truncate text-text">{{ name(person) }}</span>
                    @if (person.arrivedAt) {
                      <span class="ui-badge ui-badge-success"><span class="icon-[heroicons--check]" aria-hidden="true"></span>{{ 'tours.signup.arrived' | transloco }}</span>
                    }
                  </li>
                }
              </ul>
              <p class="ui-hint mt-2">{{ 'tours.signup.checkInHint' | transloco }}</p>
            } @else {
              <p class="text-sm text-text-muted">{{ 'tours.signup.noParticipants' | transloco }}</p>
            }
          </div>
        } @else if (started()) {
          @if (!t.isJoined) {
            <p class="text-sm text-text-muted">{{ 'tours.signup.closed' | transloco }}</p>
          }
        } @else if (t.myStatus === 'waitlist') {
          <p class="flex items-center gap-2 font-medium text-amber-700 dark:text-amber-300">
            <span class="icon-[heroicons--clock] size-5" aria-hidden="true"></span>{{ 'tours.signup.waiting' | transloco }}
          </p>
          <p class="ui-hint -mt-2">{{ 'tours.signup.waitingHint' | transloco }}</p>
          <button type="button" class="ui-btn-soft self-start" [disabled]="busy()" (click)="leave()">{{ 'tours.signup.leaveWaitlist' | transloco }}</button>
        } @else if (t.isJoined) {
          <p class="flex items-center gap-2 font-medium text-emerald-700 dark:text-emerald-300">
            <span class="icon-[heroicons--check-circle] size-5" aria-hidden="true"></span>{{ 'tours.signup.joined' | transloco }}
          </p>
          <button type="button" class="ui-btn-soft self-start" [disabled]="busy()" (click)="leave()">{{ 'tours.signup.leave' | transloco }}</button>
        } @else {
          <button type="button" class="py-2.5 text-base" [class]="full() ? 'ui-btn-soft' : 'ui-btn-primary'" [disabled]="busy()" (click)="join()">
            <span [class]="(full() ? 'icon-[heroicons--clock]' : 'icon-[heroicons--hand-raised]') + ' size-5'" aria-hidden="true"></span>{{ (full() ? 'tours.signup.joinWaitlist' : 'tours.signup.join') | transloco }}
          </button>
          <p class="ui-hint -mt-2">{{ (full() ? 'tours.signup.waitlistHint' : 'tours.signup.joinHint') | transloco }}</p>
        }
        @if (actionError(); as key) {
          <p class="ui-error" role="alert">{{ key | transloco }}</p>
        }

        @if (t.canRate || t.myRating) {
          <form class="flex flex-col gap-3 border-t border-border pt-4" novalidate (submit)="rate($event)">
            <div>
              <h3 class="font-semibold text-text">{{ 'tours.signup.rate.title' | transloco }}</h3>
              <p class="text-sm text-text-muted">{{ 'tours.signup.rate.hint' | transloco }}</p>
            </div>
            <app-rating-input [label]="'tours.signup.rate.yours' | transloco" [formField]="rateForm.rating" />
            <textarea class="ui-input" rows="3" [placeholder]="'tours.signup.rate.textPlaceholder' | transloco" [formField]="rateForm.text"></textarea>
            <div class="flex flex-wrap items-center gap-3">
              <button type="submit" class="ui-btn-primary" [disabled]="busy() || rateModel().rating === 0 || !t.canRate">{{ 'tours.signup.rate.submit' | transloco }}</button>
              @if (rateSaved()) {
                <span class="text-sm text-emerald-700 dark:text-emerald-300" role="status">{{ 'tours.signup.rate.saved' | transloco }}</span>
              }
              @if (rateError()) {
                <span class="ui-error" role="alert">{{ 'tours.signup.rate.error' | transloco }}</span>
              }
            </div>
          </form>
        }

        @if (!started() && (t.isJoined || t.isOrganizer)) {
          <app-telegram-notice />
        }

        @if (t.organizer && !t.isOrganizer) {
          <button type="button" class="ui-btn-ghost self-start" [disabled]="busy()" (click)="write()">
            <span class="icon-[heroicons--chat-bubble-left-right] size-5" aria-hidden="true"></span>{{ 'chat.writeOrganizer' | transloco }}
          </button>
        }
      </section>
    }
  `,
})
export class TourSignup {
  // The anonymous view of the tour (server-rendered).
  readonly tour = input.required<Tour>();
  // A place was taken or given up, or a rating saved: the page reloads
  // the tour and its reviews.
  readonly changed = output<void>();

  private readonly auth = inject(AuthStore);
  private readonly analytics = inject(Analytics);
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  // The signed-in visitor's view; reset when the tour changes.
  private readonly personal = linkedSignal<string, Tour | null>({
    source: () => this.tour().id,
    computation: () => null,
  });
  protected readonly item = computed(() => this.personal() ?? this.tour());

  // When the group meets, as an ISO time in Bishkek.
  private readonly startsAt = computed(() => {
    const t = this.item();
    return `${t.startDate}T${t.meetingTime || DEFAULT_MEETING_TIME}:00+06:00`;
  });
  protected readonly when = computed(() => formatInBishkek(this.lang(), this.startsAt(), WHEN));
  protected readonly started = computed(() => new Date(this.startsAt()).getTime() <= Date.now());
  protected readonly full = computed(() => {
    const t = this.item();
    return t.capacity !== null && t.taken >= t.capacity;
  });
  protected readonly filled = computed(() => {
    const t = this.item();
    return t.capacity ? Math.min(100, (t.taken / t.capacity) * 100) : 0;
  });
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
    // Signed in (now or later): read the visitor's own view.
    if (isPlatformBrowser(inject(PLATFORM_ID))) {
      void this.auth.ensureSession();
      effect(() => {
        const id = this.tour().id;
        if (this.auth.user()) void this.loadPersonal(id);
      });
    }
  }

  protected name(person: TourParticipant): string {
    return personName(person);
  }

  protected async join(): Promise<void> {
    if (!(await this.auth.ensureSession())) {
      await this.navigate(`/login?redirect=${encodeURIComponent(this.router.url)}`);
      return;
    }
    const id = this.tour().id;
    if (
      (await this.act(this.http.post<Tour>(`/tours/${id}/join`, {}))) &&
      this.personal()?.myStatus !== 'waitlist'
    ) {
      this.analytics.track('join_tour', { tour_id: id });
    }
  }

  // Opens the chat with the organizer (signing in first).
  protected async write(): Promise<void> {
    const organizer = this.item().organizer;
    if (!organizer) return;
    if (!(await this.auth.ensureSession())) {
      await this.navigate(`/login?redirect=${encodeURIComponent(this.router.url)}`);
      return;
    }
    this.busy.set(true);
    this.actionError.set(null);
    try {
      const chat = await openChat(this.http, { partnerId: organizer.id });
      await this.navigate(`/messages/${chat.id}`);
    } catch {
      this.actionError.set('chat.openError');
    } finally {
      this.busy.set(false);
    }
  }

  protected async leave(): Promise<void> {
    const waiting = this.item().myStatus === 'waitlist';
    if (!waiting && !(await this.confirm({ title: 'tours.signup.leaveConfirm' }))) return;
    const id = this.tour().id;
    if ((await this.act(this.http.delete<Tour>(`/tours/${id}/join`))) && !waiting) {
      this.analytics.track('leave_tour', { tour_id: id });
    }
  }

  protected async rate(event: Event): Promise<void> {
    event.preventDefault();
    const { rating, text } = this.rateModel();
    if (rating === 0) return;
    this.busy.set(true);
    this.rateSaved.set(false);
    this.rateError.set(false);
    try {
      const tour = await firstValueFrom(
        this.http.put<Tour>(`/tours/${this.tour().id}/rating`, {
          rating,
          text: text.trim() || null,
        }),
      );
      this.personal.set(tour);
      this.rateSaved.set(true);
      this.changed.emit();
    } catch {
      this.rateError.set(true);
    } finally {
      this.busy.set(false);
    }
  }

  private async loadPersonal(id: string): Promise<void> {
    try {
      const tour = await firstValueFrom(
        this.http.get<Tour>(`/tours/by-id/${encodeURIComponent(id)}`, { transferCache: false }),
      );
      if (id === this.tour().id) this.personal.set(tour);
    } catch {
      // The anonymous view stays.
    }
  }

  // Join or leave: the API answers with the updated tour. Whether the API
  // took it.
  private async act(request: Observable<Tour>): Promise<boolean> {
    this.busy.set(true);
    this.actionError.set(null);
    try {
      this.personal.set(await firstValueFrom(request));
      this.changed.emit();
      return true;
    } catch (error) {
      this.actionError.set(
        error instanceof HttpErrorResponse && error.status === 409
          ? 'tours.signup.errors.conflict'
          : 'tours.signup.errors.generic',
      );
      void this.loadPersonal(this.tour().id);
      return false;
    } finally {
      this.busy.set(false);
    }
  }
}
