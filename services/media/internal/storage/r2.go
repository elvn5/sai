// Package storage hands out presigned R2 upload URLs and deletes objects.
// File bytes go straight from the browser to R2 and never pass through
// this service. The bucket's CORS policy must allow PUT from the site's
// origins (see the comment in apps/api/src/storage/storage.service.ts).
package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/nodepath"
)

const uploadURLExpiry = 10 * time.Minute

// ErrNotConfigured: a required R2 setting is missing.
var ErrNotConfigured = errors.New("storage service is not configured")

type PresignedUpload struct {
	UploadURL string `json:"uploadUrl"`
	Key       string `json:"key"`
	PublicURL string `json:"publicUrl"`
}

type R2 struct {
	cfg config.Config

	once      sync.Once
	client    *s3.Client
	presigner *s3.PresignClient
	clientErr error
}

func NewR2(cfg config.Config) *R2 {
	return &R2{cfg: cfg}
}

// PresignUpload returns a short-lived URL the browser can PUT the file to.
// The object key is "<folder>/<uuid><ext>", as the NestJS API generated.
//
// Errors: ErrNotConfigured when the bucket or public URL is missing;
// anything else means the URL couldn't be signed (incl. missing
// credentials, which NestJS also reported as a signing failure).
func (r *R2) PresignUpload(ctx context.Context, filename, contentType, folder string) (PresignedUpload, error) {
	if r.cfg.R2BucketName == "" {
		return PresignedUpload{}, ErrNotConfigured
	}
	key := objectKey(filename, folder)

	if err := r.init(); err != nil {
		return PresignedUpload{}, err
	}
	req, err := r.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.cfg.R2BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(uploadURLExpiry))
	if err != nil {
		return PresignedUpload{}, fmt.Errorf("presign %q: %w", key, err)
	}

	if r.cfg.R2PublicURL == "" {
		return PresignedUpload{}, ErrNotConfigured
	}
	return PresignedUpload{
		UploadURL: req.URL,
		Key:       key,
		PublicURL: strings.TrimSuffix(r.cfg.R2PublicURL, "/") + "/" + key,
	}, nil
}

// Delete removes an object. ErrNotConfigured when the bucket is missing.
func (r *R2) Delete(ctx context.Context, key string) error {
	if r.cfg.R2BucketName == "" {
		return ErrNotConfigured
	}
	if err := r.init(); err != nil {
		return err
	}
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.cfg.R2BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}

// KeyFromPublicURL returns the object key for a URL under R2_PUBLIC_URL.
// Anything else (e.g. a legacy /uploads/... path) isn't in the bucket.
func (r *R2) KeyFromPublicURL(publicURL string) (string, bool) {
	base := strings.TrimSuffix(r.cfg.R2PublicURL, "/")
	if base == "" || !strings.HasPrefix(publicURL, base+"/") {
		return "", false
	}
	key := strings.TrimPrefix(publicURL, base+"/")
	return key, key != ""
}

func (r *R2) init() error {
	r.once.Do(func() {
		c := r.cfg
		if c.R2AccountID == "" || c.R2AccessKeyID == "" || c.R2SecretAccessKey == "" {
			r.clientErr = errors.New("R2 credentials are not fully configured (R2_ACCOUNT_ID / R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY)")
			return
		}
		endpoint := c.R2Endpoint
		if endpoint == "" {
			endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.R2AccountID)
		}
		r.client = s3.New(s3.Options{
			Region:       "auto",
			BaseEndpoint: aws.String(endpoint),
			Credentials:  credentials.NewStaticCredentialsProvider(c.R2AccessKeyID, c.R2SecretAccessKey, ""),
			UsePathStyle: c.R2Endpoint != "",
		})
		r.presigner = s3.NewPresignClient(r.client)
	})
	return r.clientErr
}

func objectKey(filename, folder string) string {
	key := uuid.NewString() + strings.ToLower(nodepath.Extname(filename))
	if folder != "" {
		return folder + "/" + key
	}
	return key
}
