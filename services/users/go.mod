module github.com/elvn5/sai/services/users

go 1.26.2

require (
	github.com/elvn5/sai/services/pkg v0.0.0
	github.com/elvn5/sai/services/proto v0.0.0
	github.com/jackc/pgx/v5 v5.11.0
	google.golang.org/grpc v1.83.2
)

require (
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/pressly/goose/v3 v3.28.0 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260831171406-18b4a7587f8a // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/elvn5/sai/services/pkg => ../pkg

replace github.com/elvn5/sai/services/proto => ../proto
