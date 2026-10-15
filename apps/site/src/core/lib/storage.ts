// localStorage that never throws (private windows, blocked site data).
// Browser only: callers run in afterNextRender or event handlers.
export function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Not saved: the setting lasts until the page is closed.
  }
}
