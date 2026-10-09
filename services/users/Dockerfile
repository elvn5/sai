# Built from the repository root: docker build -f services/users/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src/services
ENV CGO_ENABLED=0 GOWORK=off

COPY services/pkg/go.mod services/pkg/go.sum pkg/
COPY services/proto/go.mod services/proto/go.sum proto/
COPY services/users/go.mod services/users/go.sum users/
RUN cd users && go mod download

COPY services/pkg pkg
COPY services/proto proto
COPY services/users users
RUN cd users && go build -trimpath -ldflags="-s -w" -o /out/users ./cmd/users

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/users /users
EXPOSE 8084 9083
ENTRYPOINT ["/users"]
