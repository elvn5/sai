// Uploaded files come back as absolute URLs (R2) or as paths the API
// serves (/uploads/…).
export function resolveMediaUrl(apiBaseUrl: string, url: string | null): string {
  if (!url) return '';
  return /^https?:\/\//.test(url) ? url : `${apiBaseUrl}${url}`;
}
