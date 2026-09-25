// Most uploaded media is served relative to our own API (/uploads/...) and
// needs the API base URL prefixed, but R2-backed uploads store their full
// public URL directly — this tells the two apart at render time.
export function resolveMediaUrl(apiBaseUrl: string, url: string): string {
  return /^https?:\/\//.test(url) ? url : `${apiBaseUrl}${url}`;
}
