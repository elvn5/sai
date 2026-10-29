import { todayInBishkek } from '@/core/lib/format';
import { emptyToNull } from '@/core/lib/listing-form';
import { AVALANCHE_RISKS, TRACTIONS, TRAIL_STATES } from '@/core/lib/trail-options';
import { FieldErrors } from '@/ui/field-errors';
import { type IconOption, IconPicker } from '@/ui/icon-picker';
import { PhotoUploader } from '@/ui/photo-uploader';
import { SelectField, injectOptions } from '@/ui/select-field';
import { DIALOG_DATA, DialogRef } from '@angular/cdk/dialog';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormField, form, max, maxLength, min, submit, validate } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// Reports may describe the last 30 days.
const REPORT_WINDOW_DAYS = 30;
const MAX_PHOTOS = 6;

export interface ConditionReportData {
  trailId: string;
}

interface ReportModel {
  observedOn: string;
  trailState: string;
  traction: string;
  avalancheRisk: string;
  snowLine: number | null;
  reached: string;
  text: string;
  photos: string[];
}

function daysBack(days: number): string {
  return todayInBishkek(new Date(Date.now() - days * 86_400_000));
}

// "What did the route look like?" — a quick report anyone signed in files
// after a trip. Closes with true once saved.
@Component({
  selector: 'app-condition-report-dialog',
  imports: [FieldErrors, FormField, IconPicker, PhotoUploader, SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="ui-dialog flex flex-col gap-5" novalidate (submit)="send($event)">
      <div>
        <h2 class="text-lg font-bold text-text">{{ 'trails.conditions.formTitle' | transloco }}</h2>
        <p class="text-sm text-text-muted">{{ 'trails.conditions.formHint' | transloco }}</p>
      </div>
      <label class="ui-field">
        {{ 'trails.conditions.date' | transloco }}
        <input class="ui-input" type="date" [min]="earliest" [max]="today" [formField]="reportForm.observedOn" />
        <app-field-errors [state]="reportForm.observedOn()" />
      </label>
      <div class="ui-field">
        <span>{{ 'trails.conditions.trailState' | transloco }}</span>
        <app-icon-picker [options]="stateOptions" [formField]="reportForm.trailState" />
      </div>
      <div class="ui-field">
        <span>{{ 'trails.conditions.traction' | transloco }}</span>
        <app-icon-picker [options]="tractionOptions" [formField]="reportForm.traction" />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="ui-field">
          <span id="report-avalanche">{{ 'trails.conditions.avalanche' | transloco }}</span>
          <app-select-field [options]="riskOptions()" labelledBy="report-avalanche" [formField]="reportForm.avalancheRisk" />
        </div>
        <label class="ui-field">
          {{ 'trails.conditions.snowLine' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [placeholder]="'trails.conditions.snowLinePlaceholder' | transloco" [formField]="reportForm.snowLine" />
          <app-field-errors [state]="reportForm.snowLine()" />
        </label>
      </div>
      <label class="ui-field">
        {{ 'trails.conditions.reached' | transloco }}
        <input class="ui-input" [placeholder]="'trails.conditions.reachedPlaceholder' | transloco" [formField]="reportForm.reached" />
        <app-field-errors [state]="reportForm.reached()" />
      </label>
      <label class="ui-field">
        {{ 'trails.conditions.text' | transloco }}
        <textarea class="ui-input" rows="3" [placeholder]="'trails.conditions.textPlaceholder' | transloco" [formField]="reportForm.text"></textarea>
        <app-field-errors [state]="reportForm.text()" />
      </label>
      <div class="ui-field">
        <span>{{ 'trails.conditions.photos' | transloco }}</span>
        <app-photo-uploader folder="conditions" [limit]="maxPhotos" [formField]="reportForm.photos" />
      </div>
      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'trails.conditions.saveError' | transloco }}</p>
      }
      <div class="flex justify-end gap-2">
        <button type="button" class="ui-btn-ghost" (click)="ref.close()">{{ 'trails.cancel' | transloco }}</button>
        <button type="submit" class="ui-btn-primary" [disabled]="reportForm().submitting()">{{ 'trails.conditions.submit' | transloco }}</button>
      </div>
    </form>
  `,
})
export class ConditionReportDialog {
  protected readonly ref = inject<DialogRef<boolean>>(DialogRef);
  private readonly data = inject<ConditionReportData>(DIALOG_DATA);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);

  protected readonly today = todayInBishkek();
  protected readonly earliest = daysBack(REPORT_WINDOW_DAYS);
  protected readonly riskOptions = injectOptions(() => [
    { value: '', key: 'trails.conditions.avalancheUnknown' },
    ...AVALANCHE_RISKS.map((risk) => ({ value: risk, key: `trails.avalanche.${risk}` })),
  ]);
  protected readonly maxPhotos = MAX_PHOTOS;
  protected readonly stateOptions: IconOption[] = TRAIL_STATES.map((option) => ({
    ...option,
    label: this.transloco.translate(`trails.state.${option.value}`),
  }));
  protected readonly tractionOptions: IconOption[] = TRACTIONS.map((option) => ({
    ...option,
    label: this.transloco.translate(`trails.traction.${option.value}`),
  }));
  protected readonly failed = signal(false);

  private readonly model = signal<ReportModel>({
    observedOn: this.today,
    trailState: 'clear',
    traction: 'none',
    avalancheRisk: '',
    snowLine: null,
    reached: '',
    text: '',
    photos: [],
  });
  protected readonly reportForm = form(this.model, (path) => {
    const message = this.transloco.translate('trails.form.checkFields');
    validate(path.observedOn, (field) => {
      const date = field.value();
      return date >= this.earliest && date <= this.today ? null : { kind: 'range', message };
    });
    min(path.snowLine, 500, { message });
    max(path.snowLine, 7500, { message });
    maxLength(path.reached, 120);
    maxLength(path.text, 2000);
  });

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    await submit(this.reportForm, async () => {
      const value = this.model();
      try {
        await firstValueFrom(
          this.http.post(`/trails/${this.data.trailId}/conditions`, {
            observedOn: value.observedOn,
            trailState: value.trailState,
            traction: value.traction,
            avalancheRisk: value.avalancheRisk || null,
            snowLine: value.snowLine || null,
            reached: emptyToNull(value.reached),
            text: emptyToNull(value.text),
            photos: value.photos,
          }),
        );
        this.ref.close(true);
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
