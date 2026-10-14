import { RenderMode, type ServerRoute } from '@angular/ssr';

// How each route renders (see docs/angular-migration-plan.md, section 3).
// Public pages render on the server for every request, anonymously, so
// nginx can cache the HTML; account pages will be RenderMode.Client and
// static pages RenderMode.Prerender as they arrive.
export const serverRoutes: ServerRoute[] = [{ path: '**', renderMode: RenderMode.Server }];
