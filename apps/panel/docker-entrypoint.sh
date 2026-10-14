#!/bin/sh
# Writes the panel's runtime settings (see src/shared/config/app-config.ts)
# from the container's environment; nginx's entrypoint runs this on start.
set -eu
: "${API_BASE_URL:?API_BASE_URL is required}"
printf '{\n  "apiBaseUrl": "%s"\n}\n' "$API_BASE_URL" > /usr/share/nginx/html/config.json
