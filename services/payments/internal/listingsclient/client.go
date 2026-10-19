// Package listingsclient calls the listings service's internal gRPC API.
package listingsclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	listingsv1 "github.com/elvn5/sai/services/proto/gen/go/listings/v1"
)

const callTimeout = 5 * time.Second

var ErrNotFound = errors.New("not found")

type Tour struct {
	ID, Slug, Locale, Title string
	Price                   *int
	StartDate, EndDate      string
	UserID                  *string
	IsPublished             bool
	// Places for travelers; nil when unlimited.
	Capacity      *int
	CoverImageURL *string
}

type Client struct {
	conn   *grpc.ClientConn
	client listingsv1.ListingsServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("listings gRPC client: %w", err)
	}
	return &Client{conn: conn, client: listingsv1.NewListingsServiceClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Tour(ctx context.Context, id string) (Tour, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.client.GetTour(ctx, &listingsv1.GetTourRequest{Id: id})
	if status.Code(err) == codes.NotFound {
		return Tour{}, ErrNotFound
	}
	if err != nil {
		return Tour{}, fmt.Errorf("get tour: %w", err)
	}
	t := res.GetTour()
	tour := Tour{ID: t.GetId(), Slug: t.GetSlug(), Locale: t.GetLocale(), Title: t.GetTitle(),
		StartDate: t.GetStartDate(), EndDate: t.GetEndDate(), UserID: t.UserId, IsPublished: t.GetIsPublished(),
		CoverImageURL: t.CoverImageUrl}
	if t.Capacity != nil {
		capacity := int(t.GetCapacity())
		tour.Capacity = &capacity
	}
	if t.Price != nil {
		price := int(t.GetPrice())
		tour.Price = &price
	}
	return tour, nil
}
