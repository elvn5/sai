# Built from the repository root: docker build -f apps/site/Dockerfile .
FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /repo
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY apps/site/package.json apps/site/package.json
COPY packages/tokens packages/tokens
RUN pnpm install --frozen-lockfile --filter @app/site...
COPY apps/site apps/site
RUN pnpm --filter @app/site build

# The server bundle includes its dependencies: no node_modules at runtime.
# Settings come from the environment at start (API_BASE_URL, SITE_URL,
# TELEGRAM_BOT_USERNAME), so one image serves staging and production.
FROM node:24-alpine AS runtime
ENV NODE_ENV=production PORT=3000
WORKDIR /app
COPY --from=build /repo/apps/site/dist/site ./dist
USER node
EXPOSE 3000
CMD ["node", "dist/server/server.mjs"]
