import { LEGAL, LEGAL_DOCUMENTS, type LegalDocument } from '@/core/legal/documents';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Links to the legal documents (offer, privacy, refunds), in a row.
@Component({
  selector: 'app-legal-links',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <ul class="flex flex-wrap gap-x-4 gap-y-1">
      @for (doc of docs(); track doc) {
        <li><a [appSiteLink]="path(doc)" [class]="linkClass()">{{ 'legal.' + doc + '.title' | transloco }}</a></li>
      }
    </ul>
  `,
})
export class LegalLinks {
  // A document to leave out (the page it's shown on).
  readonly except = input<LegalDocument | null>(null);
  readonly linkClass = input('hover:underline');

  protected readonly docs = computed(() => LEGAL_DOCUMENTS.filter((doc) => doc !== this.except()));

  protected path(doc: LegalDocument): string {
    return LEGAL[doc].path;
  }
}
