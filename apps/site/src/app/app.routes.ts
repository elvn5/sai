import { requireAuth, requirePartner } from '@/core/auth/guards';
import { localizedRoutes } from '@/core/i18n/i18n';
import { LEGAL, LEGAL_DOCUMENTS } from '@/core/legal/documents';
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
    path: 'trails',
    loadComponent: () => import('@/pages/trails/trails-list.page').then((m) => m.TrailsListPage),
  },
  {
    path: 'trails/new',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/trails/trail-new.page').then((m) => m.TrailNewPage),
  },
  {
    path: 'trails/:slug/edit',
    canActivate: [requirePartner],
    loadComponent: () => import('@/pages/trails/trail-edit.page').then((m) => m.TrailEditPage),
  },
  {
    path: 'trails/:slug',
    loadComponent: () => import('@/pages/trails/trail.page').then((m) => m.TrailPage),
  },

  {
    path: 'outings',
    loadComponent: () => import('@/pages/outings/outings-list.page').then((m) => m.OutingsListPage),
  },
  {
    path: 'outings/new',
    canActivate: [requireAuth],
    loadComponent: () => import('@/pages/outings/outing-new.page').then((m) => m.OutingNewPage),
  },
  {
    path: 'outings/:id/edit',
    canActivate: [requireAuth],
    loadComponent: () => import('@/pages/outings/outing-edit.page').then((m) => m.OutingEditPage),
  },
  {
    path: 'outings/:id',
    loadComponent: () => import('@/pages/outings/outing.page').then((m) => m.OutingPage),
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
    path: 'payments/return',
    canActivate: [requireAuth],
    loadComponent: () =>
      import('@/pages/payments/payment-return.page').then((m) => m.PaymentReturnPage),
  },
  {
    path: 'payments/test',
    canActivate: [requireAuth],
    loadComponent: () =>
      import('@/pages/payments/payment-test.page').then((m) => m.PaymentTestPage),
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
        path: 'bookings',
        loadComponent: () =>
          import('@/pages/partner/bookings.page').then((m) => m.PartnerBookingsPage),
      },
      {
        path: 'finances',
        loadComponent: () =>
          import('@/pages/partner/finances.page').then((m) => m.PartnerFinancesPage),
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
