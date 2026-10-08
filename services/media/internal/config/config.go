// Package config holds the media service's settings.
package config

type Config struct {
	Port string `env:"PORT" envDefault:"8081"`

	// Shared with the NestJS API so existing session cookies stay valid.
	JWTSecret string `env:"JWT_SECRET,required"`

	// Local disk for /uploads (legacy images and guide attachments). Shared
	// with the NestJS API's volume during the migration.
	UploadsDir string `env:"UPLOADS_DIR" envDefault:"./uploads"`

	// Cloudflare R2. All optional so the service boots without them; the
	// storage endpoints then fail per request, as in the NestJS API.
	R2AccountID       string `env:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `env:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY"`
	R2BucketName      string `env:"R2_BUCKET_NAME"`
	R2PublicURL       string `env:"R2_PUBLIC_URL"`
	// Overrides https://<account>.r2.cloudflarestorage.com (tests, local S3).
	R2Endpoint string `env:"R2_ENDPOINT"`
}
