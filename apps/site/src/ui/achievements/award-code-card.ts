import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { encode } from 'uqr';

// The visitor's personal award code, as text and as a QR code opening the
// partner's award page with the code filled in.
@Component({
  selector: 'app-award-code-card',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .card::before {
      content: '';
      position: absolute;
      inset: -1px;
      border-radius: inherit;
      padding: 1px;
      background: linear-gradient(120deg, transparent 30%, color-mix(in srgb, var(--brand-500) 60%, transparent), transparent 70%);
      background-size: 250% 100%;
      animation: sweep 4s linear infinite;
      mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
      mask-composite: exclude;
      pointer-events: none;
    }
    @keyframes sweep { to { background-position: -250% 0; } }
    @media (prefers-reduced-motion: reduce) { .card::before { animation: none; } }
  `,
  template: `
    <div class="card ui-card relative overflow-hidden p-5">
      <div class="flex flex-col items-center gap-4">
        <svg
          [attr.viewBox]="'0 0 ' + qr().size + ' ' + qr().size"
          class="size-36 shrink-0 rounded-lg bg-white p-1"
          shape-rendering="crispEdges"
          role="img"
          [attr.aria-label]="'hallOfFame.code.qrLabel' | transloco"
        >
          <path [attr.d]="qrPath()" fill="#0f172a" />
        </svg>
        <div class="flex flex-col items-center gap-2 text-center">
          <p class="text-sm font-medium text-text-muted">{{ 'hallOfFame.code.title' | transloco }}</p>
          <button
            type="button"
            class="font-mono text-2xl font-extrabold tracking-[0.15em] whitespace-nowrap text-text transition hover:text-brand-600"
            [attr.aria-label]="('hallOfFame.code.copy' | transloco) + ': ' + formatted()"
            (click)="copy()"
          >
            {{ formatted() }}
          </button>
          <p class="text-xs text-text-muted" aria-live="polite">
            {{ (copied() ? 'hallOfFame.code.copied' : 'hallOfFame.code.hint') | transloco }}
          </p>
        </div>
      </div>
    </div>
  `,
})
export class AwardCodeCard {
  readonly code = input.required<string>();
  private readonly siteUrl = inject(APP_CONFIG).siteUrl;
  private readonly lang = injectLang();
  protected readonly copied = signal(false);

  protected readonly formatted = computed(
    () => `${this.code().slice(0, 4)}-${this.code().slice(4)}`,
  );
  private readonly awardUrl = computed(
    () =>
      `${this.siteUrl}${localizedPath(this.lang(), '/achievements/award')}?code=${encodeURIComponent(this.code())}`,
  );
  protected readonly qr = computed(() => encode(this.awardUrl(), { border: 1 }));
  // One path of 1×1 squares: far fewer DOM nodes than a rect per module.
  protected readonly qrPath = computed(() =>
    this.qr()
      .data.flatMap((row, y) => row.map((on, x) => (on ? `M${x} ${y}h1v1h-1z` : '')))
      .join(''),
  );

  protected async copy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(this.code());
      this.copied.set(true);
      setTimeout(() => this.copied.set(false), 2000);
    } catch {
      // Clipboard blocked: the code stays visible to copy by hand.
    }
  }
}
