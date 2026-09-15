import sanitizeHtml from 'sanitize-html';

/**
 * Allowlist-sanitizes rich text produced by the TipTap editor before it's
 * stored. Content created this way is rendered with v-html on the client,
 * so anything not explicitly allowed here (script tags, event handler
 * attributes, javascript: URLs, arbitrary tags) must never pass through.
 */
export function sanitizeUserContent(raw: string): string {
  return sanitizeHtml(raw, {
    allowedTags: [
      'p',
      'br',
      'strong',
      'em',
      's',
      'u',
      'code',
      'pre',
      'blockquote',
      'h2',
      'h3',
      'ul',
      'ol',
      'li',
      'a',
      'img',
    ],
    allowedAttributes: {
      a: ['href', 'target', 'rel'],
      img: ['src', 'alt', 'title'],
    },
    allowedSchemes: ['http', 'https'],
    allowedSchemesByTag: {
      img: ['http', 'https'],
    },
    transformTags: {
      a: sanitizeHtml.simpleTransform('a', { target: '_blank', rel: 'noopener noreferrer' }),
    },
  });
}
