// Sections this app serves in production. Until a section's wave is
// switched, nginx sends its paths to the Nuxt site (docs/angular-migration-
// plan.md, section 5), so links to it must load the page from the server
// instead of routing inside this app. Each wave adds its sections ("/blog"
// covers /blog and /blog/…; "/" is the home page only). Keep in step with
// the nginx locations in docs/strangler-nginx.md.
export const MIGRATED_SECTIONS: readonly string[] = [
  '/blog',
  '/guides',
  '/gallery',
  '/hall-of-fame',
];

export function isMigrated(path: string, sections: readonly string[] = MIGRATED_SECTIONS): boolean {
  return sections.some((section) =>
    section === '/' ? path === '/' : path === section || path.startsWith(`${section}/`),
  );
}
