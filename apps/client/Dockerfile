FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json ./
COPY apps/client/package.json apps/client/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/client...

FROM deps AS build
COPY apps/client apps/client
ARG NUXT_PUBLIC_API_BASE_URL
ARG NUXT_PUBLIC_SITE_URL
ENV NUXT_PUBLIC_API_BASE_URL=${NUXT_PUBLIC_API_BASE_URL}
ENV NUXT_PUBLIC_SITE_URL=${NUXT_PUBLIC_SITE_URL}
RUN pnpm --filter @app/client build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/apps/client/.output ./apps/client/.output
WORKDIR /repo/apps/client
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
