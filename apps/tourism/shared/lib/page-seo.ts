import { type UseQueryReturnType, useQuery } from '@tanstack/vue-query';
import type { Ref } from 'vue';
import { useApiClient } from '~/shared/api/client';
import type { PageSeo, SeoPage } from '~/shared/api/types';

export function usePageSeo(
  page: SeoPage,
  locale: Ref<string>,
): UseQueryReturnType<PageSeo | null, Error> {
  const api = useApiClient();

  return useQuery({
    queryKey: ['page-seo', page, locale],
    queryFn: () => api.get<PageSeo | null>(`/page-seo/${page}/${locale.value}`),
  });
}
