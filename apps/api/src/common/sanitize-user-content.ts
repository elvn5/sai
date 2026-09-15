import sanitizeHtml from 'sanitize-html';

/**
 * Turns arbitrary user-submitted text into safe HTML paragraphs. Content
 * created this way is rendered with v-html on the client, so it must never
 * carry through tags/attributes an author typed — everything is stripped to
 * plain text first, then line-broken into <p> tags.
 */
export function sanitizeUserContent(raw: string): string {
  const plainText = sanitizeHtml(raw, { allowedTags: [], allowedAttributes: {} });

  return plainText
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => `<p>${line}</p>`)
    .join('');
}
