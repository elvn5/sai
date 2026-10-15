import type { ConditionReport } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDate, personName } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import {
  FRESH_REPORT_DAYS,
  TRACTIONS,
  TRAIL_STATES,
  daysSince,
  optionIcon,
} from '@/core/lib/trail-options';
import { PersonAvatar } from '@/ui/person-avatar';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input, output } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// One condition report: who went, when, what the route was like.
@Component({
  selector: 'app-condition-card',
  imports: [NgOptimizedImage, PersonAvatar, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="ui-card p-4" [class.opacity-70]="stale()">
      <header class="flex items-center gap-3">
        <app-person-avatar [person]="report().author" [size]="36" />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-semibold text-text">{{ author() }}</p>
          <p class="text-xs text-text-muted">
            {{ 'trails.conditions.observed' | transloco: { date: observed() } }}
            @if (report().reached; as place) {
              · {{ 'trails.conditions.reachedShort' | transloco: { place: place } }}
            }
          </p>
        </div>
        @if (canDelete()) {
          <button type="button" class="ui-btn-ghost ui-btn-xs" [attr.aria-label]="'trails.conditions.delete' | transloco" (click)="delete.emit()">
            <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>
          </button>
        }
      </header>
      <ul class="mt-3 flex flex-wrap gap-2 text-xs">
        <li class="inline-flex items-center gap-1 rounded-full bg-text/5 px-2.5 py-1 text-text">
          <span [class]="stateIcon() + ' size-4'" aria-hidden="true"></span>{{ 'trails.state.' + report().trailState | transloco }}
        </li>
        <li class="inline-flex items-center gap-1 rounded-full bg-text/5 px-2.5 py-1 text-text">
          <span [class]="tractionIcon() + ' size-4'" aria-hidden="true"></span>{{ 'trails.traction.' + report().traction | transloco }}
        </li>
        @if (report().snowLine; as meters) {
          <li class="inline-flex items-center gap-1 rounded-full bg-text/5 px-2.5 py-1 text-text">
            <span class="icon-[mdi--snowflake] size-4" aria-hidden="true"></span>{{ 'trails.conditions.snowLineShort' | transloco: { meters: meters } }}
          </li>
        }
        @if (report().avalancheRisk; as risk) {
          <li class="inline-flex items-center gap-1 rounded-full px-2.5 py-1" [class]="risk === 'low' ? 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-200' : 'bg-red-500/15 text-red-800 dark:text-red-200'">
            <span class="icon-[mdi--alert-outline] size-4" aria-hidden="true"></span>{{ 'trails.avalanche.' + risk | transloco }}
          </li>
        }
      </ul>
      @if (report().text) {
        <p class="mt-3 text-sm whitespace-pre-line text-text">{{ report().text }}</p>
      }
      @if (report().photos.length > 0) {
        <div class="mt-3 flex gap-2 overflow-x-auto">
          @for (photo of report().photos; track photo; let i = $index) {
            <a [href]="media(photo)" target="_blank" rel="noopener" class="shrink-0" [attr.aria-label]="('trails.conditions.photos' | transloco) + ' ' + (i + 1)">
              <img [ngSrc]="media(photo)" width="128" height="96" alt="" class="h-24 w-32 rounded-lg object-cover" />
            </a>
          }
        </div>
      }
    </article>
  `,
})
export class ConditionCard {
  readonly report = input.required<ConditionReport>();
  readonly canDelete = input(false);
  readonly delete = output();

  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();

  protected readonly author = computed(() => personName(this.report().author));
  protected readonly stale = computed(
    () => daysSince(this.report().observedOn) > FRESH_REPORT_DAYS,
  );
  protected readonly observed = computed(() =>
    formatDate(this.lang(), this.report().observedOn, { day: 'numeric', month: 'long' }),
  );
  protected readonly stateIcon = computed(() => optionIcon(TRAIL_STATES, this.report().trailState));
  protected readonly tractionIcon = computed(() => optionIcon(TRACTIONS, this.report().traction));

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }
}
