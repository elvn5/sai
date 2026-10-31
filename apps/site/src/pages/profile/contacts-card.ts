import type { AppUser } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { FieldErrors } from '@/ui/field-errors';
import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const POLL_MS = 3000;
const WAIT_MS = 5 * 60_000;

// An account made by signing in with Telegram has no email or phone: this
// card asks for one (a reminder, nothing is blocked). The email is
// confirmed with a code; the phone is shared from the bot (Telegram
// vouches for it). If either already belongs to another account of the
// person, the two accounts become one.
@Component({
  selector: 'app-contacts-card',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (!user().email || (!user().phone && phoneUrl())) {
      <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="contacts-heading">
        <div>
          <h2 id="contacts-heading" class="font-semibold text-text">{{ 'profile.contacts.title' | transloco }}</h2>
          <p class="text-sm text-text-muted">{{ 'profile.contacts.hint' | transloco }}</p>
        </div>
        @if (!user().email && !user().phone) {
          <p class="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/40 dark:text-amber-200" role="note">
            <span class="icon-[heroicons--exclamation-triangle] mt-0.5 size-4 shrink-0" aria-hidden="true"></span>{{ 'profile.contacts.reminder' | transloco }}
          </p>
        }

        @if (!user().email) {
          @if (sentTo(); as address) {
            <form class="flex max-w-md flex-col gap-3" novalidate (submit)="confirm($event)">
              <p class="text-sm text-text">{{ 'profile.contacts.codeSent' | transloco: { email: address } }}</p>
              <label class="block">
                <span class="ui-label">{{ 'profile.contacts.code' | transloco }}</span>
                <input class="ui-input" inputmode="numeric" autocomplete="one-time-code" [formField]="cf.code" />
                <app-field-errors [state]="cf.code()" />
              </label>
              <div class="flex flex-wrap gap-2">
                <button type="submit" class="ui-btn-primary" [disabled]="cf().submitting()" [attr.aria-busy]="cf().submitting()">
                  {{ 'profile.contacts.confirm' | transloco }}
                </button>
                <button type="button" class="ui-btn-ghost" (click)="restart()">{{ 'profile.contacts.otherEmail' | transloco }}</button>
              </div>
            </form>
          } @else {
            <form class="flex max-w-md flex-col gap-3" novalidate (submit)="send($event)">
              <label class="block">
                <span class="ui-label">{{ 'profile.fields.email' | transloco }}</span>
                <input class="ui-input" type="email" autocomplete="email" placeholder="you@example.com" [formField]="ef.email" />
                <app-field-errors [state]="ef.email()" />
              </label>
              <button type="submit" class="ui-btn-soft self-start" [disabled]="ef().submitting()" [attr.aria-busy]="ef().submitting()">
                <span class="icon-[heroicons--envelope] size-4" aria-hidden="true"></span>{{ 'profile.contacts.sendCode' | transloco }}
              </button>
            </form>
          }
        }

        @if (!user().phone && phoneUrl(); as url) {
          <div class="flex flex-col gap-2">
            <p class="text-sm text-text-muted">{{ 'profile.contacts.phoneHint' | transloco }}</p>
            <a [href]="url" target="_blank" rel="noopener noreferrer" class="ui-btn-soft self-start" (click)="waitForPhone()">
              <span class="icon-[mdi--telegram] size-4 text-sky-600" aria-hidden="true"></span>{{ 'profile.contacts.addPhone' | transloco }}
            </a>
            @if (waiting()) {
              <span class="inline-flex items-center gap-2 text-sm text-text-muted" role="status">
                <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>{{ 'profile.contacts.phoneWaiting' | transloco }}
              </span>
            }
          </div>
        }

        @if (error(); as key) {
          <p class="ui-error" role="alert">{{ key | transloco }}</p>
        }
      </section>
    }
  `,
})
export class ContactsCard {
  readonly user = input.required<AppUser>();

  private readonly auth = inject(AuthStore);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly bot = inject(APP_CONFIG).telegramBotUsername;
  private readonly navigate = injectSiteNavigate();

  // The phone is shared in the bot, so only a Telegram account gets the
  // button (an account without Telegram registered with its email).
  protected readonly phoneUrl = computed(() =>
    this.bot && this.user().telegramId ? `https://t.me/${this.bot}?start=phone` : null,
  );
  // The email a code was sent to.
  protected readonly sentTo = signal<string | null>(null);
  // Translation key of a failure the fields don't show.
  protected readonly error = signal<string | null>(null);
  // Whether the page waits for the number shared in the bot.
  protected readonly waiting = signal(false);
  private waitingUntil = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
  }

  private readonly emailModel = signal({ email: '' });
  protected readonly ef = form(this.emailModel, (path) => {
    maxLength(path.email, 255);
    validateStandardSchema(
      path.email,
      z
        .string()
        .check(
          z.trim(),
          z.regex(EMAIL_PATTERN, this.transloco.translate('profile.contacts.badEmail')),
        ),
    );
  });

  private readonly codeModel = signal({ code: '' });
  protected readonly cf = form(this.codeModel, (path) => {
    maxLength(path.code, 6);
    validateStandardSchema(
      path.code,
      z.string().check(z.regex(/^\d{6}$/, this.transloco.translate('login.verify.invalidFormat'))),
    );
  });

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.ef, async () => {
      const email = this.emailModel().email.trim();
      try {
        await this.auth.addEmail(email);
        this.sentTo.set(email);
      } catch (error) {
        const status = error instanceof HttpErrorResponse ? error.status : 0;
        this.error.set(
          status === 409
            ? 'profile.contacts.emailTaken'
            : status === 429
              ? 'profile.contacts.wait'
              : 'profile.contacts.error',
        );
      }
      return undefined;
    });
  }

  protected async confirm(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.cf, async () => {
      try {
        const { merged } = await this.auth.confirmEmail(this.codeModel().code);
        this.notify.success({
          title: merged ? 'profile.contacts.merged' : 'profile.contacts.emailAdded',
        });
        this.restart();
      } catch (error) {
        const status = error instanceof HttpErrorResponse ? error.status : 0;
        this.error.set(
          status === 401
            ? 'profile.contacts.wrongCode'
            : status === 400
              ? 'profile.contacts.codeExpired'
              : status === 409
                ? 'profile.contacts.emailTaken'
                : 'profile.contacts.error',
        );
      }
      return undefined;
    });
  }

  // Re-reads the account while the visitor shares the number in the bot.
  // If the number was another account's, this Telegram account was merged
  // into it and its session is gone: signing in with Telegram again opens
  // the merged account.
  protected waitForPhone(): void {
    this.waiting.set(true);
    this.waitingUntil = Date.now() + WAIT_MS;
    this.poll();
  }

  private poll(): void {
    clearTimeout(this.timer);
    this.timer = setTimeout(async () => {
      const user = await this.auth.refresh().catch(() => undefined);
      if (user === null) {
        this.waiting.set(false);
        this.notify.info({ title: 'profile.contacts.mergedSignIn' });
        void this.navigate('/login');
      } else if (user?.phone) {
        this.waiting.set(false);
        this.notify.success({ title: 'profile.contacts.phoneAdded' });
      } else if (Date.now() < this.waitingUntil) {
        this.poll();
      } else {
        this.waiting.set(false);
      }
    }, POLL_MS);
  }

  protected restart(): void {
    this.sentTo.set(null);
    this.error.set(null);
    this.codeModel.set({ code: '' });
    this.cf().reset();
  }
}
