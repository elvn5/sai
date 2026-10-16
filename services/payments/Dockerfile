# Built from the repository root: docker build -f services/payments/Dockerfile .
FROM golang:1.27-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/payments/go.mod services/payments/go.sum payments/
RUN cd payments && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/payments payments
RUN cd payments && go build -trimpath -ldflags="-s -w" -o /out/payments ./cmd/payments

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/payments /payments
EXPOSE 8086
ENTRYPOINT ["/payments"]
