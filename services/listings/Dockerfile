# Built from the repository root: docker build -f services/listings/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/listings/go.mod services/listings/go.sum listings/
RUN cd listings && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/listings listings
RUN cd listings && go build -trimpath -ldflags="-s -w" -o /out/listings ./cmd/listings

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/listings /listings
EXPOSE 8083 9082
ENTRYPOINT ["/listings"]
