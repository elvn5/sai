// A legal document's text, in Russian. Sections become the table of
// contents; `body` is HTML (p, ul, ol, li, strong, a). Text marked with
// todo()/todoInline() is still to be written: the page highlights it.
export interface LegalSection {
  id: string;
  title: string;
  body: string;
}

export interface LegalText {
  title: string;
  // One line under the title, for search results and previews.
  description: string;
  // When this edition takes effect (YYYY-MM-DD); null while a draft.
  effectiveFrom: string | null;
  sections: LegalSection[];
}

// A paragraph still to be written: what goes there.
export function todo(hint: string): string {
  return `<p class="legal-todo">[${hint}]</p>`;
}

// The same inside a sentence or a list item.
export function todoInline(hint: string): string {
  return `<span class="legal-todo">[${hint}]</span>`;
}
