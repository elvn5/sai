FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json tsconfig.base.json ./
COPY packages/database/package.json packages/database/package.json
COPY apps/bot/package.json apps/bot/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/bot...

FROM deps AS build
COPY packages/database packages/database
COPY apps/bot apps/bot
RUN pnpm --filter @app/database... build && pnpm --filter @app/bot build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/node_modules ./node_modules
COPY --from=build /repo/packages/database ./packages/database
COPY --from=build /repo/apps/bot/node_modules ./apps/bot/node_modules
COPY --from=build /repo/apps/bot/dist ./apps/bot/dist
COPY --from=build /repo/apps/bot/package.json ./apps/bot/package.json
WORKDIR /repo/apps/bot
CMD ["node", "dist/main.js"]
