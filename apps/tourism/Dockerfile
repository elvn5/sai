FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json ./
COPY apps/tourism/package.json apps/tourism/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/tourism...

FROM deps AS build
COPY apps/tourism apps/tourism
ARG NUXT_PUBLIC_API_BASE_URL
ARG NUXT_PUBLIC_SITE_URL
ENV NUXT_PUBLIC_API_BASE_URL=${NUXT_PUBLIC_API_BASE_URL}
ENV NUXT_PUBLIC_SITE_URL=${NUXT_PUBLIC_SITE_URL}
RUN pnpm --filter @app/tourism build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/apps/tourism/.output ./apps/tourism/.output
WORKDIR /repo/apps/tourism
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
