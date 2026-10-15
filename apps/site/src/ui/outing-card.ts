import type { Outing } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek } from '@/core/lib/format';
import { LEVELS, TRANSPORTS, optionIcon } from '@/core/lib/trail-options';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// An upcoming outing: the date block, where and how, and seats left.
// `compact` stacks everything in one column, for narrow sidebars.
@Component({
  selector: 'app-outing-card',
  imports: [PersonAvatar, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <a
      [appSiteLink]="'/outings/' + outing().id"
      class="group flex rounded-card border border-border bg-surface transition hover:border-brand-400 hover:shadow-md"
      [class]="compact() ? 'gap-3 p-3' : 'gap-4 p-4'"
      [class.opacity-60]="cancelled()"
    >
      <span class="flex w-14 shrink-0 flex-col items-center justify-center rounded-xl bg-linear-to-b from-brand-600 to-brand-800 py-2 text-white">
        <span class="text-2xl leading-none font-black">{{ day() }}</span>
        <span class="text-xs uppercase">{{ month() }}</span>
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-xs text-text-muted">{{ weekdayTime() }}</span>
        <span class="block truncate font-semibold text-text group-hover:text-brand-700 dark:group-hover:text-brand-300">{{ outing().title }}</span>
        @if (outing().trail; as trail) {
          <span class="flex items-center gap-1 truncate text-sm text-text-muted">
            <span class="icon-[mdi--map-marker-path] size-4 shrink-0" aria-hidden="true"></span>{{ trail.name }}
          </span>
        }
        @if (!compact()) {
          <span class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-muted">
            <span class="inline-flex items-center gap-1">
              <span [class]="transportIcon() + ' size-4'" aria-hidden="true"></span>{{ 'outings.transport.' + outing().transport | transloco }}
            </span>
            <span class="inline-flex items-center gap-1">
              <span [class]="levelIcon() + ' size-4'" aria-hidden="true"></span>{{ 'outings.level.' + outing().level | transloco }}
            </span>
            @if (outing().costPerPerson !== null) {
              <span>
                @if (outing().costPerPerson === 0) {
                  {{ 'outings.free' | transloco }}
                } @else {
                  {{ cost() }} {{ 'listing.currency' | transloco }}
                }
              </span>
            }
          </span>
        } @else {
          <span class="mt-1 block text-xs font-semibold" [class]="seatsClass()">{{ status() | transloco: { count: left() } }}</span>
        }
      </span>
      @if (!compact()) {
        <span class="flex shrink-0 flex-col items-end justify-between gap-2">
          <app-person-avatar [person]="outing().organizer" [size]="32" />
          <span class="rounded-full px-2 py-0.5 text-xs font-semibold" [class]="seatsClass() + ' ' + seatsBackground()">{{ status() | transloco: { count: left() } }}</span>
        </span>
      }
    </a>
  `,
})
export class OutingCard {
  readonly outing = input.required<Outing>();
  readonly compact = input(false);
  private readonly lang = injectLang();

  protected readonly cancelled = computed(() => this.outing().status === 'cancelled');
  protected readonly left = computed(() => Math.max(0, this.outing().seats - this.outing().taken));
  protected readonly day = computed(() =>
    formatInBishkek(this.lang(), this.outing().startsAt, { day: 'numeric' }),
  );
  protected readonly month = computed(() =>
    formatInBishkek(this.lang(), this.outing().startsAt, { month: 'short' }),
  );
  protected readonly weekdayTime = computed(() =>
    formatInBishkek(this.lang(), this.outing().startsAt, {
      weekday: 'short',
      hour: '2-digit',
      minute: '2-digit',
    }),
  );
  protected readonly cost = computed(() =>
    formatAmount(this.lang(), this.outing().costPerPerson ?? 0),
  );
  protected readonly transportIcon = computed(() =>
    optionIcon(TRANSPORTS, this.outing().transport),
  );
  protected readonly levelIcon = computed(() => optionIcon(LEVELS, this.outing().level));
  // Translation key of the seats line.
  protected readonly status = computed(() => {
    if (this.cancelled()) return 'outings.cancelled';
    return this.left() === 0 ? 'outings.full' : 'outings.seatsLeft';
  });
  protected readonly seatsClass = computed(() =>
    this.cancelled() || this.left() === 0
      ? 'text-text-muted'
      : 'text-emerald-800 dark:text-emerald-200',
  );
  protected readonly seatsBackground = computed(() =>
    this.cancelled() || this.left() === 0 ? 'bg-text/10' : 'bg-emerald-500/15',
  );
}
