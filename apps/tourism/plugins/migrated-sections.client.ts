// Sections the Angular site (apps/site) serves in production: nginx sends
// their paths there (docs/strangler-nginx.md). Navigating to one loads the
// page from the server instead of rendering this app's old copy in place.
// Keep in step with MIGRATED_SECTIONS in apps/site/src/core/config/sections.ts.
const MIGRATED_SECTIONS = [
  '/blog',
  '/guides',
  '/gallery',
  '/hall-of-fame',
  '/hotels',
  '/local-guides',
  '/tours',
  '/trails',
  '/outings',
  '/',
];

function isMigrated(path: string): boolean {
  const unprefixed = path.replace(/^\/(ky|en|de)(?=\/|$)/, '') || '/';
  // "/" is the home page only.
  return MIGRATED_SECTIONS.some((section) =>
    section === '/'
      ? unprefixed === '/'
      : unprefixed === section || unprefixed.startsWith(`${section}/`),
  );
}

export default defineNuxtPlugin(() => {
  const router = useRouter();
  router.beforeEach((to, from) => {
    // The first page of a visit is already the server's answer.
    if (from.matched.length === 0 || !isMigrated(to.path)) return;
    window.location.assign(to.fullPath);
    return false;
  });
});
