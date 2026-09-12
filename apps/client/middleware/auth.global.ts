import { useAuthStore } from '~/stores/auth';

export default defineNuxtRouteMiddleware(async (to) => {
  const getRouteBaseName = useRouteBaseName();
  const publicPages = ['index', 'login', 'about', 'blog', 'blog-slug', 'gallery'];
  if (publicPages.includes(getRouteBaseName(to) ?? '')) return;

  const auth = useAuthStore();
  if (!auth.checked) {
    await auth.checkSession();
  }

  if (!auth.authenticated) {
    const localePath = useLocalePath();
    return navigateTo(localePath('/login'));
  }
});
