# Built from the repository root: docker build -f services/gateway/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/gateway/go.mod services/gateway/go.sum gateway/
RUN cd gateway && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/gateway gateway
RUN cd gateway && go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gateway /gateway
EXPOSE 8080
ENTRYPOINT ["/gateway"]
