import type { Review } from '@/core/api/types';
import { FieldErrors } from '@/ui/field-errors';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const MAX_LENGTH = 2000;

// The listing owner's public answer to a review: write, edit or remove it.
// `changed` fires once the API has it, so the list (and its badge) reload.
@Component({
  selector: 'app-review-reply-form',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (review().reply && !editing()) {
      <div class="rounded-lg border-l-4 border-brand-500 bg-brand-500/5 px-4 py-3">
        <p class="text-xs font-semibold text-brand-700 dark:text-brand-300">{{ 'reviews.ownerReply' | transloco }}</p>
        <p class="mt-1 text-sm whitespace-pre-line text-text">{{ review().reply?.text }}</p>
        <div class="mt-2 flex gap-2">
          <button type="button" class="ui-btn-soft ui-btn-xs" (click)="editing.set(true)">
            <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
          </button>
          <button type="button" class="ui-btn-ghost ui-btn-xs" [disabled]="removing()" (click)="remove()">
            <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>{{ 'reviews.delete' | transloco }}
          </button>
        </div>
      </div>
    } @else {
      <form class="flex flex-col gap-2" novalidate (submit)="save($event)">
        <label class="sr-only" [for]="'reply-' + review().id">{{ 'cabinet.reviews.publishReply' | transloco }}</label>
        <textarea class="ui-input" rows="3" [id]="'reply-' + review().id" [placeholder]="'cabinet.reviews.replyPlaceholder' | transloco" [formField]="replyForm.text"></textarea>
        <app-field-errors [state]="replyForm.text()" />
        @if (failed()) {
          <p class="ui-error" role="alert">{{ 'cabinet.reviews.saveError' | transloco }}</p>
        }
        <div class="flex gap-2">
          <button type="submit" class="ui-btn-primary" [disabled]="replyForm().submitting()">{{ 'cabinet.reviews.publishReply' | transloco }}</button>
          @if (review().reply) {
            <button type="button" class="ui-btn-ghost" (click)="editing.set(false)">{{ 'cabinet.reviews.cancel' | transloco }}</button>
          }
        </div>
      </form>
    }
  `,
})
export class ReviewReplyForm {
  readonly review = input.required<Review>();
  readonly changed = output();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  protected readonly editing = linkedSignal(() => !this.review().reply);
  protected readonly removing = signal(false);
  protected readonly failed = signal(false);

  private readonly model = linkedSignal(() => ({ text: this.review().reply?.text ?? '' }));
  protected readonly replyForm = form(this.model, (path) => {
    validateStandardSchema(
      path,
      z.object({
        text: z
          .string()
          .check(z.trim(), z.minLength(1, this.transloco.translate('cabinet.reviews.emptyReply'))),
      }),
    );
    maxLength(path.text, MAX_LENGTH);
  });

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    await submit(this.replyForm, async () => {
      try {
        await firstValueFrom(
          this.http.put(`/reviews/${this.review().id}/reply`, { text: this.model().text.trim() }),
        );
        this.editing.set(false);
        this.changed.emit();
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }

  protected async remove(): Promise<void> {
    if (!window.confirm(this.transloco.translate('cabinet.reviews.deleteConfirm'))) return;
    this.removing.set(true);
    try {
      await firstValueFrom(this.http.delete(`/reviews/${this.review().id}/reply`));
      this.changed.emit();
    } finally {
      this.removing.set(false);
    }
  }
}
