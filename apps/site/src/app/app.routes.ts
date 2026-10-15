import { localizedRoutes } from '@/core/i18n/i18n';
import type { Routes } from '@angular/router';

// Pages, served under every language prefix (see localizedRoutes).
const pages: Routes = [
  { path: '', loadComponent: () => import('@/pages/home/home.page').then((m) => m.HomePage) },
  {
    path: '**',
    loadComponent: () => import('@/pages/not-found/not-found.page').then((m) => m.NotFoundPage),
  },
];

export const routes: Routes = localizedRoutes(pages);
