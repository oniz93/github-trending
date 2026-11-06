package database

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNormalizeEndpoint(t *testing.T) {
	cases := map[string]string{
		"seaweedfs:8333":         "http://seaweedfs:8333",
		"http://seaweedfs:8333":  "http://seaweedfs:8333",
		"https://s3.example.com": "https://s3.example.com",
		" seaweedfs:8333/ ":      "http://seaweedfs:8333",
	}
	for in, want := range cases {
		if got := normalizeEndpoint(in); got != want {
			t.Errorf("normalizeEndpoint(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestS3ConnectionRoundTrip runs against a real S3-compatible server, e.g.:
//
//	docker run -d -p 8333:8333 -e AWS_ACCESS_KEY_ID=test -e AWS_SECRET_ACCESS_KEY=testsecret \
//	  chrislusf/seaweedfs server -dir=/data -s3
//	S3_TEST_ENDPOINT=localhost:8333 S3_TEST_ACCESS_KEY=test S3_TEST_SECRET_KEY=testsecret go test ./internal/database/
func TestS3ConnectionRoundTrip(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}
	bucket := fmt.Sprintf("test-%d", time.Now().UnixNano())

	conn, err := NewS3Connection(endpoint, os.Getenv("S3_TEST_ACCESS_KEY"), os.Getenv("S3_TEST_SECRET_KEY"), bucket)
	if err != nil {
		t.Fatalf("NewS3Connection: %v", err)
	}
	// Connecting again must accept the now-existing bucket.
	if _, err := NewS3Connection(endpoint, os.Getenv("S3_TEST_ACCESS_KEY"), os.Getenv("S3_TEST_SECRET_KEY"), bucket); err != nil {
		t.Fatalf("NewS3Connection on existing bucket: %v", err)
	}

	ctx := context.Background()
	content := []byte("# Hello\n\nREADME content")
	if err := conn.UploadFile(ctx, "readmes/1.md", bytes.NewReader(content), int64(len(content)), "text/markdown"); err != nil {
		t.Fatalf("UploadFile (seekable): %v", err)
	}
	got, err := conn.GetFile(ctx, "readmes/1.md")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("GetFile = %q, want %q", got, content)
	}

	// Non-seekable readers are buffered before upload.
	if err := conn.UploadFile(ctx, "readmes/2.md", strings.NewReader("two"), 3, "text/markdown"); err != nil {
		t.Fatalf("UploadFile (reader): %v", err)
	}
	if got, err := conn.GetFile(ctx, "readmes/2.md"); err != nil || string(got) != "two" {
		t.Fatalf("GetFile(2) = %q, %v", got, err)
	}

	if _, err := conn.GetFile(ctx, "readmes/missing.md"); err == nil {
		t.Fatal("GetFile on missing object: expected error")
	}

	// Wrong credentials must be rejected.
	if _, err := NewS3Connection(endpoint, os.Getenv("S3_TEST_ACCESS_KEY"), "wrong-secret", bucket); err == nil {
		t.Fatal("NewS3Connection with wrong secret: expected error")
	}
}
