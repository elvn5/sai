import type { ParticipantForm, ProfileParticipantForm } from '@/core/api/types';
import { Notify } from '@/core/notify/notify';
import { ParticipantFormView } from '@/ui/participant-form';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, linkedSignal, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// The participant's form in the profile: filled in once, it goes with
// every tour the traveler signs up for (organizers see it). Optional.
@Component({
  selector: 'app-participant-form-card',
  imports: [ParticipantFormView, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="participant-form-heading">
      <div class="flex flex-wrap items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <h2 id="participant-form-heading" class="font-semibold text-text">{{ 'participantForm.title' | transloco }}</h2>
          <p class="text-sm text-text-muted">{{ 'participantForm.profileWhy' | transloco }}</p>
        </div>
        @if (current() && !editing()) {
          <button type="button" class="ui-btn-soft ui-btn-xs" (click)="editing.set(true)">
            <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'participantForm.edit' | transloco }}
          </button>
        }
      </div>

      @if (data.error()) {
        <p class="ui-error" role="alert">{{ 'participantForm.loadError' | transloco }}</p>
      } @else if (!data.hasValue()) {
        <div class="ui-skeleton h-16 rounded-lg" aria-hidden="true"></div>
      } @else if (editing()) {
        <app-participant-form [value]="current()" [busy]="busy()" [failed]="failed()" (saved)="save($event)" (cancelled)="editing.set(false)" />
      } @else if (current(); as f) {
        <dl class="grid gap-x-4 gap-y-1.5 text-sm sm:grid-cols-[auto_1fr]">
          <dt class="text-text-muted">{{ 'participantForm.emergencyName' | transloco }}</dt>
          <dd class="text-text">{{ f.emergencyName }}, {{ f.emergencyPhone }}</dd>
          <dt class="text-text-muted">{{ 'participantForm.experience' | transloco }}</dt>
          <dd class="text-text">{{ 'participantForm.levels.' + f.experience + '.title' | transloco }}</dd>
          @if (f.health) {
            <dt class="text-text-muted">{{ 'participantForm.health' | transloco }}</dt>
            <dd class="whitespace-pre-line text-text">{{ f.health }}</dd>
          }
          @if (f.notes) {
            <dt class="text-text-muted">{{ 'participantForm.notes' | transloco }}</dt>
            <dd class="whitespace-pre-line text-text">{{ f.notes }}</dd>
          }
        </dl>
      } @else {
        <p class="text-sm text-text-muted">{{ 'participantForm.empty' | transloco }}</p>
        <button type="button" class="ui-btn-primary self-start" (click)="editing.set(true)">
          <span class="icon-[heroicons--clipboard-document-list] size-4" aria-hidden="true"></span>{{ 'participantForm.fill' | transloco }}
        </button>
      }
    </section>
  `,
})
export class ParticipantFormCard {
  private readonly http = inject(HttpClient);
  private readonly notify = inject(Notify);

  protected readonly data = httpResource<ProfileParticipantForm>(() => '/tours/participant-form');
  // The saved form, replaced by what saving answers.
  protected readonly current = linkedSignal<ParticipantForm | null>(
    () => this.data.value()?.form ?? null,
  );
  protected readonly editing = signal(false);
  protected readonly busy = signal(false);
  // Translation key of a failed save.
  protected readonly failed = signal<string | null>(null);

  protected async save(form: ParticipantForm): Promise<void> {
    this.busy.set(true);
    this.failed.set(null);
    try {
      const saved = await firstValueFrom(
        this.http.put<ProfileParticipantForm>('/tours/participant-form', { form }),
      );
      this.current.set(saved.form);
      this.editing.set(false);
      this.notify.success({ title: 'participantForm.saved' });
    } catch {
      this.failed.set('participantForm.saveError');
    } finally {
      this.busy.set(false);
    }
  }
}
