import { requirePartner } from '@/core/auth/guards';
import { localizedRoutes } from '@/core/i18n/i18n';
import type { Routes } from '@angular/router';

// Pages, served under every language prefix (see localizedRoutes). Forms
// come before the slug routes they share a prefix with.
const pages: Routes = [
  { path: '', loadComponent: () => import('@/pages/home/home.page').then((m) => m.HomePage) },

  {
    path: 'blog',
    loadComponent: () => import('@/pages/blog/blog-list.page').then((m) => m.BlogListPage),
  },
  {
    path: 'blog/new',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/blog/blog-new.page').then((m) => m.BlogNewPage),
  },
  {
    path: 'blog/edit/:id',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/blog/blog-edit.page').then((m) => m.BlogEditPage),
  },
  {
    path: 'blog/:slug',
    loadComponent: () => import('@/pages/blog/blog-post.page').then((m) => m.BlogPostPage),
  },

  {
    path: 'guides',
    loadComponent: () => import('@/pages/guides/guides-list.page').then((m) => m.GuidesListPage),
  },
  {
    path: 'guides/new',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/guides/guide-new.page').then((m) => m.GuideNewPage),
  },
  {
    path: 'guides/:slug',
    loadComponent: () => import('@/pages/guides/guide.page').then((m) => m.GuidePage),
  },

  {
    path: 'gallery',
    loadComponent: () => import('@/pages/gallery/gallery.page').then((m) => m.GalleryPage),
  },
  {
    path: 'hall-of-fame',
    loadComponent: () =>
      import('@/pages/hall-of-fame/hall-of-fame.page').then((m) => m.HallOfFamePage),
  },

  {
    path: '**',
    loadComponent: () => import('@/pages/not-found/not-found.page').then((m) => m.NotFoundPage),
  },
];

export const routes: Routes = localizedRoutes(pages);
