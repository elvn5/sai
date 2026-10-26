import { requireAuth, requirePartner } from '@/core/auth/guards';
import { localizedRoutes } from '@/core/i18n/i18n';
import { LEGAL, LEGAL_DOCUMENTS } from '@/core/legal/documents';
import { prefetchTour } from '@/pages/tours/tour-prefetch';
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
    path: 'gallery',
    loadComponent: () => import('@/pages/gallery/gallery.page').then((m) => m.GalleryPage),
  },
  {
    path: 'hall-of-fame',
    loadComponent: () =>
      import('@/pages/hall-of-fame/hall-of-fame.page').then((m) => m.HallOfFamePage),
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
    resolve: { prefetched: prefetchTour },
    loadComponent: () => import('@/pages/tours/tour.page').then((m) => m.TourPage),
  },

  {
    path: 'organizers/:id',
    loadComponent: () => import('@/pages/organizers/organizer.page').then((m) => m.OrganizerPage),
  },

  ...LEGAL_DOCUMENTS.map((doc) => ({
    path: LEGAL[doc].path.slice(1),
    data: { doc },
    loadComponent: () => import('@/pages/legal/legal.page').then((m) => m.LegalPage),
  })),

  {
    path: 'login',
    loadComponent: () => import('@/pages/login/login.page').then((m) => m.LoginPage),
  },
  {
    path: 'profile',
    canActivate: [requireAuth],
    loadComponent: () => import('@/pages/profile/profile.page').then((m) => m.ProfilePage),
  },
  {
    path: 'achievements/award',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/achievements/award.page').then((m) => m.AwardPage),
  },
  {
    path: 'partner',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/partner/partner-layout').then((m) => m.PartnerLayout),
    children: [
      {
        path: '',
        loadComponent: () =>
          import('@/pages/partner/dashboard.page').then((m) => m.PartnerDashboardPage),
      },
      {
        path: 'stats',
        loadComponent: () => import('@/pages/partner/stats.page').then((m) => m.PartnerStatsPage),
      },
      {
        path: 'listings',
        loadComponent: () =>
          import('@/pages/partner/listings.page').then((m) => m.PartnerListingsPage),
      },
      {
        path: 'reviews',
        loadComponent: () =>
          import('@/pages/partner/reviews.page').then((m) => m.PartnerReviewsPage),
      },
      {
        path: 'tours/:id',
        loadComponent: () =>
          import('@/pages/partner/tour-edit.page').then((m) => m.PartnerTourEditPage),
      },
    ],
  },

  {
    path: '**',
    loadComponent: () => import('@/pages/not-found/not-found.page').then((m) => m.NotFoundPage),
  },
];

export const routes: Routes = localizedRoutes(pages);
