import { LEGAL, LEGAL_DOCUMENTS, publishedLegalPaths } from '@/core/legal/documents';

describe('legal documents', () => {
  it('lists only finished documents for search engines', () => {
    const drafts = LEGAL_DOCUMENTS.filter((doc) => LEGAL[doc].draft).map((doc) => LEGAL[doc].path);
    for (const path of drafts) expect(publishedLegalPaths()).not.toContain(path);
    expect(publishedLegalPaths().length + drafts.length).toBe(LEGAL_DOCUMENTS.length);
  });
});
