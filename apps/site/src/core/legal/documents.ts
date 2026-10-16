// The site's legal documents. Each has a page at its path (pages/legal) and
// a Russian text in pages/legal/texts; other languages show the Russian
// text with a note. A draft is shown with a notice, kept out of search
// (noindex) and out of the sitemap — clear `draft` once the final text is in.
export const LEGAL_DOCUMENTS = ['offer', 'privacy', 'refunds'] as const;
export type LegalDocument = (typeof LEGAL_DOCUMENTS)[number];

export interface LegalDocumentInfo {
  // Unprefixed path ("/offer").
  path: string;
  draft: boolean;
}

export const LEGAL: Record<LegalDocument, LegalDocumentInfo> = {
  offer: { path: '/offer', draft: true },
  privacy: { path: '/privacy', draft: true },
  refunds: { path: '/refunds', draft: true },
};

// Paths of the documents that are ready for search engines.
export function publishedLegalPaths(): string[] {
  return LEGAL_DOCUMENTS.filter((doc) => !LEGAL[doc].draft).map((doc) => LEGAL[doc].path);
}
