FROM node:25-alpine AS base
RUN corepack enable
WORKDIR /repo

FROM base AS deps
COPY pnpm-workspace.yaml package.json tsconfig.base.json ./
COPY apps/bot/package.json apps/bot/package.json
RUN pnpm install --frozen-lockfile=false --filter @app/bot...

FROM deps AS build
COPY apps/bot apps/bot
RUN pnpm --filter @app/bot build

FROM base AS runtime
ENV NODE_ENV=production
COPY --from=build /repo/node_modules ./node_modules
COPY --from=build /repo/apps/bot/node_modules ./apps/bot/node_modules
COPY --from=build /repo/apps/bot/dist ./apps/bot/dist
COPY --from=build /repo/apps/bot/package.json ./apps/bot/package.json
# The users service's gRPC contract, loaded at runtime (dist/users-grpc
# resolves it four levels up, at /repo/services/proto).
COPY services/proto/users services/proto/users
WORKDIR /repo/apps/bot
CMD ["node", "dist/main.js"]
