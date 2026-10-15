import { requireAuth, requirePartner } from '@/core/auth/guards';
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
    path: 'hotels',
    loadComponent: () => import('@/pages/hotels/hotels-list.page').then((m) => m.HotelsListPage),
  },
  {
    path: 'hotels/new',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/hotels/hotel-new.page').then((m) => m.HotelNewPage),
  },
  {
    path: 'hotels/:slug/edit',
    canActivate: [requireAuth],
    loadComponent: () => import('@/pages/hotels/hotel-edit.page').then((m) => m.HotelEditPage),
  },
  {
    path: 'hotels/:slug',
    loadComponent: () => import('@/pages/hotels/hotel.page').then((m) => m.HotelPage),
  },

  {
    path: 'local-guides',
    loadComponent: () =>
      import('@/pages/local-guides/local-guides-list.page').then((m) => m.LocalGuidesListPage),
  },
  {
    path: 'local-guides/new',
    canActivate: [requirePartner],
    loadComponent: () =>
      import('@/pages/local-guides/local-guide-new.page').then((m) => m.LocalGuideNewPage),
  },
  {
    path: 'local-guides/:slug/edit',
    canActivate: [requireAuth],
    loadComponent: () =>
      import('@/pages/local-guides/local-guide-edit.page').then((m) => m.LocalGuideEditPage),
  },
  {
    path: 'local-guides/:slug',
    loadComponent: () =>
      import('@/pages/local-guides/local-guide.page').then((m) => m.LocalGuidePage),
  },

  {
    path: 'tours',
    loadComponent: () => import('@/pages/tours/tours-list.page').then((m) => m.ToursListPage),
  },
  {
    path: 'tours/new',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/tours/tour-new.page').then((m) => m.TourNewPage),
  },
  {
    path: 'tours/:slug',
    loadComponent: () => import('@/pages/tours/tour.page').then((m) => m.TourPage),
  },

  {
    path: '**',
    loadComponent: () => import('@/pages/not-found/not-found.page').then((m) => m.NotFoundPage),
  },
];

export const routes: Routes = localizedRoutes(pages);
