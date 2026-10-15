import { requireAdmin } from '@/core/auth/auth';
import type { Routes } from '@angular/router';

// The sections of the Nuxt admin (apps/admin), same paths.
export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('@/pages/login/login.page').then((m) => m.LoginPage),
  },
  {
    path: '',
    loadComponent: () => import('@/layout/admin-layout').then((m) => m.AdminLayout),
    canActivate: [requireAdmin],
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'users' },
      {
        path: 'support',
        loadComponent: () => import('@/pages/support/support.page').then((m) => m.SupportPage),
      },
      {
        path: 'users',
        loadComponent: () => import('@/pages/users/users.page').then((m) => m.UsersPage),
      },
      {
        path: 'partners',
        loadComponent: () => import('@/pages/partners/partners.page').then((m) => m.PartnersPage),
      },
      {
        path: 'achievements',
        loadComponent: () =>
          import('@/pages/achievements/achievements.page').then((m) => m.AchievementsPage),
      },
      {
        path: 'trails',
        loadComponent: () => import('@/pages/trails/trails.page').then((m) => m.TrailsPage),
      },
      {
        path: 'payments',
        loadComponent: () => import('@/pages/payments/payments.page').then((m) => m.PaymentsPage),
      },
      {
        path: 'blog',
        loadComponent: () => import('@/pages/blog/blog-list.page').then((m) => m.BlogListPage),
      },
      {
        path: 'blog/:id',
        loadComponent: () => import('@/pages/blog/blog-edit.page').then((m) => m.BlogEditPage),
      },
      {
        path: 'seo',
        loadComponent: () => import('@/pages/seo/seo.page').then((m) => m.SeoSettingsPage),
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
