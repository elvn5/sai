import { requireAdmin } from '@/features/auth/auth';
import type { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('@/pages/login/login.page').then((m) => m.LoginPage),
  },
  {
    path: '',
    canActivate: [requireAdmin],
    loadComponent: () => import('@/pages/home/home.page').then((m) => m.HomePage),
  },
  { path: '**', redirectTo: '' },
];
