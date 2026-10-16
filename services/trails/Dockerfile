# Built from the repository root: docker build -f services/trails/Dockerfile .
FROM golang:1.27-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/trails/go.mod services/trails/go.sum trails/
RUN cd trails && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/trails trails
RUN cd trails && go build -trimpath -ldflags="-s -w" -o /out/trails ./cmd/trails

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/trails /trails
EXPOSE 8085
ENTRYPOINT ["/trails"]
