FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json tsconfig.base.json ./
COPY packages/database/package.json packages/database/package.json
COPY apps/api/package.json apps/api/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/api...

FROM deps AS build
COPY packages/database packages/database
COPY apps/api apps/api
RUN pnpm --filter @app/database... build && pnpm --filter @app/api build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/node_modules ./node_modules
COPY --from=build /repo/packages/database ./packages/database
COPY --from=build /repo/apps/api/node_modules ./apps/api/node_modules
COPY --from=build /repo/apps/api/dist ./apps/api/dist
COPY --from=build /repo/apps/api/package.json ./apps/api/package.json
WORKDIR /repo/apps/api
EXPOSE 3001
CMD ["node", "dist/main.js"]
