# Built from the repository root: docker build -f services/media/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
# nodynamic: gen2brain/webp must use its bundled pure-Go codec, not try to
# dlopen a system libwebp (which also makes the binary need a dynamic loader).
ENV CGO_ENABLED=0 GOWORK=off GOFLAGS=-tags=nodynamic

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/media/go.mod services/media/go.sum media/
RUN cd media && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/media media
RUN cd media && go build -trimpath -ldflags="-s -w" -o /out/media ./cmd/media

# Runs as root (not :nonroot) because it writes to the uploads volume that
# the NestJS API created as root.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/media /media
EXPOSE 8081 9081
ENTRYPOINT ["/media"]
