# Built from the repository root: docker build -f services/bot/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/bot/go.mod services/bot/go.sum bot/
RUN cd bot && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/bot bot
RUN cd bot && go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

# No ports: the bot polls Telegram and calls the users service.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bot /bot
ENTRYPOINT ["/bot"]
