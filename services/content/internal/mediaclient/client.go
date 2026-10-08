// Package mediaclient calls the media service's internal gRPC API.
package mediaclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	mediav1 "github.com/elvn5/sai/services/proto/gen/go/media/v1"
)

const callTimeout = 5 * time.Second

type Client struct {
	conn   *grpc.ClientConn
	client mediav1.MediaServiceClient
}

// New connects lazily to addr (host:port). Traffic stays on the private
// Docker network, hence plaintext.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("media gRPC client: %w", err)
	}
	return &Client{conn: conn, client: mediav1.NewMediaServiceClient(conn)}, nil
}

// DeleteObject removes a stored file by its public URL. Files outside the
// R2 bucket (legacy /uploads paths) are left alone by media.
func (c *Client) DeleteObject(ctx context.Context, publicURL string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	_, err := c.client.DeleteObject(ctx, &mediav1.DeleteObjectRequest{PublicUrl: publicURL})
	return err
}

func (c *Client) Close() error { return c.conn.Close() }
