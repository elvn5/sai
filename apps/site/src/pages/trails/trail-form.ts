import type { TrackPoint, Trail, TrailSegment } from '@/core/api/types';
import { UploadApi } from '@/core/api/upload';
import { injectLang } from '@/core/i18n/current-lang';
import { parseGpx } from '@/core/lib/gpx';
import { emptyToNull } from '@/core/lib/listing-form';
import { AREAS, GEAR, GRADES, KINDS, SEASONS, formatGrade } from '@/core/lib/trail-options';
import { ElevationProfile } from '@/pages/trails/elevation-profile';
import { FieldErrors } from '@/ui/field-errors';
import { IconMultiPicker, type IconOption, IconPicker } from '@/ui/icon-picker';
import { PhotoUploader } from '@/ui/photo-uploader';
import { RichTextEditor } from '@/ui/rich-text-editor';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import {
  FormField,
  applyEach,
  form,
  max,
  maxLength,
  min,
  submit,
  validateStandardSchema,
} from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const MAX_PHOTOS = 12;

// What the API takes to create or update a route.
export interface TrailPayload {
  name: string;
  summary: string | null;
  description: string;
  area: string;
  kind: string;
  grade: string;
  seasons: string[];
  gear: string[];
  summitElevation: number | null;
  startElevation: number | null;
  elevationGain: number | null;
  distanceKm: number | null;
  durationHours: number | null;
  days: number;
  segments: TrailSegment[];
  gettingThere: string | null;
  track: TrackPoint[];
  gpxUrl: string | null;
  coverImageUrl: string | null;
  photos: string[];
  isPublished: boolean;
}

interface SegmentModel {
  from: string;
  to: string;
  hours: number | null;
}

interface TrailFormModel {
  name: string;
  summary: string;
  area: string;
  grade: string;
  kind: string;
  summitElevation: number | null;
  startElevation: number | null;
  elevationGain: number | null;
  distanceKm: number | null;
  durationHours: number | null;
  days: number | null;
  segments: SegmentModel[];
  seasons: string[];
  gear: string[];
  gettingThere: string;
  description: string;
  photos: string[];
  isPublished: boolean;
}

function toModel(trail: Trail | undefined): TrailFormModel {
  return {
    name: trail?.name ?? '',
    summary: trail?.summary ?? '',
    area: trail?.area ?? 'ala_archa',
    grade: trail?.grade ?? 'trail',
    kind: trail?.kind ?? 'hike',
    summitElevation: trail?.summitElevation ?? null,
    startElevation: trail?.startElevation ?? null,
    elevationGain: trail?.elevationGain ?? null,
    distanceKm: trail?.distanceKm ?? null,
    durationHours: trail?.durationHours ?? null,
    days: trail?.days ?? 1,
    segments: trail?.segments.map((segment) => ({ ...segment })) ?? [],
    seasons: trail?.seasons ?? ['summer'],
    gear: trail?.gear ?? [],
    gettingThere: trail?.gettingThere ?? '',
    description: trail?.description ?? '',
    // The cover first: the uploader treats the first photo as the cover.
    photos: trail
      ? [
          ...(trail.coverImageUrl ? [trail.coverImageUrl] : []),
          ...trail.photos.filter((photo) => photo !== trail.coverImageUrl),
        ]
      : [],
    isPublished: trail?.isPublished ?? true,
  };
}

// A partner's route: creates one, or edits `trail`. The GPX track is read
// in the browser (map and profile) and the file itself uploaded for
// download. `save` sends it and moves on.
@Component({
  selector: 'app-trail-form',
  imports: [
    ElevationProfile,
    FieldErrors,
    FormField,
    IconMultiPicker,
    IconPicker,
    PhotoUploader,
    RichTextEditor,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-8" novalidate (submit)="send($event)">
      <fieldset class="flex flex-col gap-5">
        <label class="ui-field">
          {{ 'trails.form.name' | transloco }} *
          <input class="ui-input py-2.5 text-base" [placeholder]="'trails.form.namePlaceholder' | transloco" [formField]="trailForm.name" />
          <app-field-errors [state]="trailForm.name()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.summary' | transloco }}
          <input class="ui-input" [placeholder]="'trails.form.summaryPlaceholder' | transloco" [formField]="trailForm.summary" />
          <app-field-errors [state]="trailForm.summary()" />
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="ui-field">
            {{ 'trails.form.area' | transloco }}
            <select class="ui-input" [formField]="trailForm.area">
              @for (area of areas; track area) {
                <option [value]="area">{{ 'trails.area.' + area | transloco }}</option>
              }
            </select>
          </label>
          <label class="ui-field">
            {{ 'trails.form.grade' | transloco }}
            <select class="ui-input" [formField]="trailForm.grade">
              @for (grade of grades; track grade) {
                <option [value]="grade">
                  @if (grade === 'trail') { {{ 'trails.grade.trail' | transloco }} } @else { {{ gradeLabel(grade) }} }
                </option>
              }
            </select>
          </label>
        </div>
        <div class="ui-field">
          <span>{{ 'trails.form.kind' | transloco }}</span>
          <app-icon-picker [options]="kindOptions()" [formField]="trailForm.kind" />
        </div>
      </fieldset>

      <section class="ui-card flex flex-col gap-4 p-4" aria-labelledby="gpx-title">
        <h2 id="gpx-title" class="font-semibold text-text">{{ 'trails.form.trackTitle' | transloco }}</h2>
        <p class="text-sm text-text-muted">{{ 'trails.form.trackHint' | transloco }}</p>
        <div class="flex flex-wrap items-center gap-3">
          <label class="ui-btn-primary cursor-pointer focus-within:ring-2 focus-within:ring-brand-500/40">
            <input type="file" accept=".gpx,application/gpx+xml" class="sr-only" (change)="onGpx($event)" />
            <span class="icon-[mdi--upload] size-4" aria-hidden="true"></span>{{ 'trails.form.uploadGpx' | transloco }}
          </label>
          @if (gpxUploading()) {
            <span class="text-sm text-text-muted">{{ 'trails.form.uploading' | transloco }}</span>
          } @else if (track().length > 0) {
            <span class="text-sm text-text">
              {{ 'trails.form.trackPoints' | transloco: { count: track().length } }}
              <button type="button" class="ml-2 text-brand-700 hover:underline dark:text-brand-300" (click)="clearTrack()">{{ 'trails.form.removeTrack' | transloco }}</button>
            </span>
          }
        </div>
        @if (gpxError()) {
          <p class="ui-error" role="alert">{{ 'trails.form.gpxError' | transloco }}</p>
        }
        @if (track().length > 1) {
          <app-elevation-profile [track]="track()" />
        }
      </section>

      <div class="grid gap-4 sm:grid-cols-3">
        <label class="ui-field">
          {{ 'trails.form.summit' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="trailForm.summitElevation" />
          <app-field-errors [state]="trailForm.summitElevation()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.start' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="trailForm.startElevation" />
          <app-field-errors [state]="trailForm.startElevation()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.gain' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="trailForm.elevationGain" />
          <app-field-errors [state]="trailForm.elevationGain()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.distance' | transloco }}
          <input class="ui-input" type="number" step="0.1" inputmode="decimal" [formField]="trailForm.distanceKm" />
          <app-field-errors [state]="trailForm.distanceKm()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.duration' | transloco }}
          <input class="ui-input" type="number" step="0.5" inputmode="decimal" [formField]="trailForm.durationHours" />
          <app-field-errors [state]="trailForm.durationHours()" />
        </label>
        <label class="ui-field">
          {{ 'trails.form.days' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="trailForm.days" />
          <app-field-errors [state]="trailForm.days()" />
        </label>
      </div>

      <section class="flex flex-col gap-3" aria-labelledby="segments-title">
        <div class="flex items-center justify-between">
          <h2 id="segments-title" class="font-semibold text-text">{{ 'trails.form.segments' | transloco }}</h2>
          <button type="button" class="ui-btn-soft ui-btn-xs" (click)="addSegment()">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'trails.form.addSegment' | transloco }}
          </button>
        </div>
        @for (segment of trailForm.segments; track segment; let i = $index) {
          <div class="grid grid-cols-[1fr_1fr_5rem_auto] items-start gap-2">
            <input class="ui-input" [placeholder]="'trails.form.from' | transloco" [attr.aria-label]="'trails.form.from' | transloco" [formField]="segment.from" />
            <input class="ui-input" [placeholder]="'trails.form.to' | transloco" [attr.aria-label]="'trails.form.to' | transloco" [formField]="segment.to" />
            <input class="ui-input" type="number" step="0.5" inputmode="decimal" [placeholder]="'trails.form.hours' | transloco" [attr.aria-label]="'trails.form.hours' | transloco" [formField]="segment.hours" />
            <button type="button" class="ui-btn-ghost px-2 py-2" [attr.aria-label]="'trails.form.removeTrack' | transloco" (click)="removeSegment(i)">
              <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>
            </button>
          </div>
        } @empty {
          <p class="text-sm text-text-muted">{{ 'trails.form.segmentsHint' | transloco }}</p>
        }
      </section>

      <div class="ui-field">
        <span>{{ 'trails.form.seasons' | transloco }}</span>
        <app-icon-multi-picker [options]="seasonOptions()" [formField]="trailForm.seasons" />
      </div>
      <div class="ui-field">
        <span>{{ 'trails.form.gear' | transloco }}</span>
        <app-icon-multi-picker [options]="gearOptions()" [formField]="trailForm.gear" />
      </div>
      <label class="ui-field">
        {{ 'trails.form.gettingThere' | transloco }}
        <textarea class="ui-input" rows="3" [placeholder]="'trails.form.gettingTherePlaceholder' | transloco" [formField]="trailForm.gettingThere"></textarea>
        <app-field-errors [state]="trailForm.gettingThere()" />
      </label>
      <div class="ui-field">
        <span>{{ 'trails.form.description' | transloco }}</span>
        <app-rich-text-editor [placeholder]="'trails.form.descriptionPlaceholder' | transloco" [formField]="trailForm.description" />
        <app-field-errors [state]="trailForm.description()" />
      </div>
      <div class="ui-field">
        <span>{{ 'trails.form.photos' | transloco }}</span>
        <span class="ui-hint">{{ 'trails.form.photosHint' | transloco }}</span>
        <app-photo-uploader folder="trails" [limit]="maxPhotos" [coverBadge]="true" [formField]="trailForm.photos" />
      </div>
      <label class="flex items-center gap-3 text-sm text-text">
        <input type="checkbox" class="size-4 accent-brand-600" [formField]="trailForm.isPublished" />
        {{ 'trails.form.publish' | transloco }}
      </label>

      @if (failed(); as key) {
        <p class="ui-error" role="alert">{{ key | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="trailForm().submitting() || gpxUploading()">{{ submitLabel() }}</button>
    </form>
  `,
})
export class TrailForm {
  readonly trail = input<Trail | undefined>(undefined);
  readonly submitLabel = input.required<string>();
  readonly save = input.required<(payload: TrailPayload) => Promise<void>>();

  private readonly transloco = inject(TranslocoService);
  private readonly uploads = inject(UploadApi);
  protected readonly areas = AREAS;
  protected readonly grades = GRADES;
  protected readonly maxPhotos = MAX_PHOTOS;
  private readonly lang = injectLang();
  // Options read the language at render: labels follow the switcher.
  protected readonly kindOptions = computed(() => this.options(KINDS, 'trails.kind'));
  protected readonly seasonOptions = computed(() => this.options(SEASONS, 'trails.season'));
  protected readonly gearOptions = computed(() => this.options(GEAR, 'trails.gear'));
  protected readonly gradeLabel = formatGrade;

  protected readonly model = linkedSignal(() => toModel(this.trail()));
  // Read from the GPX file; replaced by a new upload.
  protected readonly track = linkedSignal<TrackPoint[]>(() => this.trail()?.track ?? []);
  private readonly gpxUrl = linkedSignal<string | null>(() => this.trail()?.gpxUrl ?? null);
  protected readonly gpxUploading = signal(false);
  protected readonly gpxError = signal(false);
  // Translation key of the submit error.
  protected readonly failed = signal<string | null>(null);

  protected readonly trailForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const message = t('outings.form.checkFields');
    validateStandardSchema(
      path,
      z.object({ name: z.string().check(z.trim(), z.minLength(1, t('login.required'))) }),
    );
    maxLength(path.name, 160);
    maxLength(path.summary, 300);
    maxLength(path.gettingThere, 2000);
    maxLength(path.description, 50000);
    min(path.summitElevation, 500, { message });
    max(path.summitElevation, 7500, { message });
    min(path.startElevation, 300, { message });
    max(path.startElevation, 7500, { message });
    min(path.elevationGain, 0, { message });
    max(path.elevationGain, 7000, { message });
    min(path.distanceKm, 0, { message });
    min(path.durationHours, 0, { message });
    min(path.days, 1, { message });
    max(path.days, 60, { message });
    applyEach(path.segments, (segment) => {
      maxLength(segment.from, 100);
      maxLength(segment.to, 100);
      min(segment.hours, 0, { message });
    });
  });

  protected async onGpx(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    this.gpxError.set(false);
    try {
      const points = parseGpx(await file.text());
      if (points.length < 2) throw new Error('empty-track');
      this.track.set(points);
      // Heights the visitor hasn't filled in come from the track.
      const heights = points.map((point) => point[2]).filter((ele): ele is number => ele !== null);
      if (heights.length > 0)
        this.model.update((value) => ({
          ...value,
          startElevation: value.startElevation ?? heights[0] ?? null,
          summitElevation: value.summitElevation ?? Math.max(...heights),
        }));
      this.gpxUploading.set(true);
      this.gpxUrl.set((await this.uploads.uploadFile(file)).url);
    } catch {
      this.gpxError.set(true);
    } finally {
      this.gpxUploading.set(false);
    }
  }

  protected clearTrack(): void {
    this.track.set([]);
    this.gpxUrl.set(null);
  }

  // A new leg starts where the last one ended.
  protected addSegment(): void {
    this.model.update((value) => ({
      ...value,
      segments: [...value.segments, { from: value.segments.at(-1)?.to ?? '', to: '', hours: 1 }],
    }));
  }

  protected removeSegment(index: number): void {
    this.model.update((value) => ({
      ...value,
      segments: value.segments.filter((_, i) => i !== index),
    }));
  }

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(null);
    await submit(this.trailForm, async () => {
      const value = this.model();
      try {
        await this.save()({
          name: value.name.trim(),
          summary: emptyToNull(value.summary),
          description: value.description,
          area: value.area,
          kind: value.kind,
          grade: value.grade,
          seasons: value.seasons,
          gear: value.gear,
          summitElevation: value.summitElevation,
          startElevation: value.startElevation,
          elevationGain: value.elevationGain,
          distanceKm: value.distanceKm,
          durationHours: value.durationHours,
          days: value.days || 1,
          // Half-filled legs are dropped.
          segments: value.segments
            .filter((s) => s.from.trim() && s.to.trim() && (s.hours ?? 0) > 0)
            .map((s) => ({ from: s.from.trim(), to: s.to.trim(), hours: s.hours ?? 0 })),
          gettingThere: emptyToNull(value.gettingThere),
          track: this.track(),
          gpxUrl: this.gpxUrl(),
          coverImageUrl: value.photos[0] ?? null,
          photos: value.photos,
          isPublished: value.isPublished,
        });
      } catch (error) {
        this.failed.set(
          (error as { status?: number }).status === 400
            ? 'outings.form.checkFields'
            : 'trails.form.saveError',
        );
      }
      return undefined;
    });
  }

  private options(options: { value: string; icon: string }[], prefix: string): IconOption[] {
    this.lang();
    return options.map((option) => ({
      ...option,
      label: this.transloco.translate(`${prefix}.${option.value}`),
    }));
  }
}
