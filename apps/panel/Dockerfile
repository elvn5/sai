# Built from the repository root: docker build -f apps/panel/Dockerfile .
FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /repo
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY apps/panel/package.json apps/panel/package.json
COPY packages/tokens packages/tokens
RUN pnpm install --frozen-lockfile --filter @app/panel...
COPY apps/panel apps/panel
RUN pnpm --filter @app/panel build

# Static files behind nginx (unprivileged, port 3000). config.json is
# written from API_BASE_URL when the container starts, so one image serves
# every environment.
FROM nginxinc/nginx-unprivileged:stable-alpine AS runtime
COPY apps/panel/nginx.conf /etc/nginx/conf.d/default.conf
COPY apps/panel/docker-entrypoint.sh /docker-entrypoint.d/40-panel-config.sh
COPY --from=build --chown=101:101 /repo/apps/panel/dist/panel/browser /usr/share/nginx/html
EXPOSE 3000
