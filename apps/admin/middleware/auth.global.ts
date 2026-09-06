import { useAuthStore } from '~/stores/auth';

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path === '/login') return;

  const auth = useAuthStore();
  if (!auth.checked) {
    await auth.checkSession();
  }

  if (!auth.authenticated) {
    return navigateTo('/login');
  }
});
