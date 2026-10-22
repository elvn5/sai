import type { FirebaseWebConfig } from '@/core/config/app-config';

// What the site needs from Firebase Analytics.
export interface AnalyticsBackend {
  log(name: string, params: Record<string, unknown>): void;
  setEnabled(enabled: boolean): void;
}

// Loads Firebase Analytics (its own lazy chunk, fetched only once a visitor
// allows analytics). null where the browser can't run it (no cookies or
// IndexedDB). Page views are sent by the site (core/analytics), not by
// Firebase on load: the app changes pages without reloading.
export async function loadFirebaseAnalytics(
  config: FirebaseWebConfig,
): Promise<AnalyticsBackend | null> {
  const [{ initializeApp }, analytics] = await Promise.all([
    import('firebase/app'),
    import('firebase/analytics'),
  ]);
  if (!(await analytics.isSupported())) return null;
  const app = initializeApp(config);
  const instance = analytics.initializeAnalytics(app, { config: { send_page_view: false } });
  return {
    log: (name, params) => analytics.logEvent(instance, name, params),
    setEnabled: (enabled) => analytics.setAnalyticsCollectionEnabled(instance, enabled),
  };
}
