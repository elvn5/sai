FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json ./
COPY apps/admin/package.json apps/admin/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/admin...

FROM deps AS build
COPY apps/admin apps/admin
ARG NUXT_PUBLIC_API_BASE_URL
ENV NUXT_PUBLIC_API_BASE_URL=${NUXT_PUBLIC_API_BASE_URL}
RUN pnpm --filter @app/admin build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/apps/admin/.output ./apps/admin/.output
WORKDIR /repo/apps/admin
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
