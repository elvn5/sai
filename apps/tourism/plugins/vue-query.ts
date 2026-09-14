import { QueryClient, VueQueryPlugin, dehydrate, hydrate } from '@tanstack/vue-query';
import type { DehydratedState, VueQueryPluginOptions } from '@tanstack/vue-query';

export default defineNuxtPlugin((nuxtApp) => {
  const vueQueryState = useState<DehydratedState | null>('vue-query');

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000,
        gcTime: 10 * 60 * 1000,
        refetchOnWindowFocus: false,
        retry: 1,
      },
    },
  });
  const options: VueQueryPluginOptions = { queryClient };

  nuxtApp.vueApp.use(VueQueryPlugin, options);

  if (import.meta.server) {
    nuxtApp.hooks.hook('app:rendered', () => {
      vueQueryState.value = dehydrate(queryClient);
    });
  } else {
    nuxtApp.hooks.hook('app:created', () => {
      if (vueQueryState.value) {
        hydrate(queryClient, vueQueryState.value);
      }
    });
  }
});
