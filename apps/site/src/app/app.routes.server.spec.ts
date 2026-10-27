import { serverRoutes } from '@/app/app.routes.server';
import { RenderMode } from '@angular/ssr';

describe('serverRoutes', () => {
  it('renders forms in the browser in every language, the rest on the server', () => {
    const client = serverRoutes
      .filter((route) => route.renderMode === RenderMode.Client)
      .map((route) => route.path);
    expect(client).toContain('blog/new');
    expect(client).toContain('en/blog/edit/:id');
    expect(client).toContain('de/tours/new');
    expect(client).toContain('trails/new');
    expect(client).toContain('ky/trails/:slug/edit');
    expect(serverRoutes.at(-1)).toMatchObject({ path: '**', renderMode: RenderMode.Server });
  });
});
