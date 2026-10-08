package grpcserver_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/grpcserver"
	"github.com/elvn5/sai/services/media/internal/storage"
	mediav1 "github.com/elvn5/sai/services/proto/gen/go/media/v1"
)

func TestDeleteObject(t *testing.T) {
	var deleted []string
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleted = append(deleted, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer s3.Close()

	r2 := storage.NewR2(config.Config{
		R2AccountID: "acc", R2AccessKeyID: "k", R2SecretAccessKey: "s",
		R2BucketName: "photos", R2PublicURL: "https://cdn.example.com", R2Endpoint: s3.URL,
	})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	mediav1.RegisterMediaServiceServer(srv, &grpcserver.Server{R2: r2, Log: slog.New(slog.DiscardHandler)})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	client := mediav1.NewMediaServiceClient(conn)

	res, err := client.DeleteObject(context.Background(), &mediav1.DeleteObjectRequest{PublicUrl: "https://cdn.example.com/gallery/a.webp"})
	if err != nil || !res.GetDeleted() {
		t.Fatalf("R2 url: deleted=%v err=%v", res.GetDeleted(), err)
	}
	res, err = client.DeleteObject(context.Background(), &mediav1.DeleteObjectRequest{PublicUrl: "/uploads/legacy.webp"})
	if err != nil || res.GetDeleted() {
		t.Fatalf("legacy url: deleted=%v err=%v", res.GetDeleted(), err)
	}
	if len(deleted) != 1 || deleted[0] != "DELETE /photos/gallery/a.webp" {
		t.Fatalf("S3 calls = %v", deleted)
	}
}
