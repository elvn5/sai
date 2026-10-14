// Package config holds the bot's settings.
package config

type Config struct {
	BotToken string `env:"BOT_TOKEN,required"`
	// Overrides the Bot API server (a local Bot API server, or tests).
	TelegramAPIURL string `env:"TELEGRAM_API_URL"`

	// users service: accounts, Telegram linking, support contacts.
	UsersGRPCAddr string `env:"USERS_GRPC_ADDR,required"`

	// URL of the Ermekten Mini App (the public site). When set, the bot
	// exposes it via the chat menu button and a keyboard button.
	MiniAppURL string `env:"MINI_APP_URL"`
}
