import { useAuthStore } from '~/stores/auth';

export default defineNuxtRouteMiddleware(async (to) => {
  const publicPages = ['/', '/login', '/about'];
  if (publicPages.includes(to.path)) return;

  const auth = useAuthStore();
  if (!auth.checked) {
    await auth.checkSession();
  }

  if (!auth.authenticated) {
    return navigateTo('/login');
  }
});
