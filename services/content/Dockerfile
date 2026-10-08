# Built from the repository root: docker build -f services/content/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/content/go.mod services/content/go.sum content/
RUN cd content && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/content content
RUN cd content && go build -trimpath -ldflags="-s -w" -o /out/content ./cmd/content

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/content /content
EXPOSE 8082
ENTRYPOINT ["/content"]
