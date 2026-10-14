module github.com/elvn5/sai/services/bot

go 1.26.7

replace github.com/elvn5/sai/services/pkg => ../pkg

replace github.com/elvn5/sai/services/proto => ../proto

require (
	github.com/elvn5/sai/services/pkg v0.0.0
	github.com/elvn5/sai/services/proto v0.0.0
	github.com/mymmrac/telego v1.12.1
	google.golang.org/grpc v1.84.0
)

require (
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.3 // indirect
	github.com/bytedance/sonic/loader v0.5.2 // indirect
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/grbit/go-json v0.11.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.73.0 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260720211330-0afa2a65878a // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
