package config

import "testing"

func TestS3SettingsPrefersS3Set(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ROOT_USER", "minio-user")
	t.Setenv("MINIO_ROOT_PASSWORD", "minio-pass")
	t.Setenv("S3_ENDPOINT", "seaweedfs:8333")
	t.Setenv("S3_ACCESS_KEY", "s3-key")
	t.Setenv("S3_SECRET_KEY", "s3-secret")

	e, a, s := s3Settings()
	if e != "seaweedfs:8333" || a != "s3-key" || s != "s3-secret" {
		t.Fatalf("got %q %q %q", e, a, s)
	}
}

func TestS3SettingsLegacyNotMixed(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ROOT_USER", "minio-user")
	t.Setenv("MINIO_ROOT_PASSWORD", "minio-pass")
	t.Setenv("S3_ENDPOINT", "")
	// S3 credentials alone (e.g. only consumed by the object store container)
	// must not be combined with the legacy endpoint.
	t.Setenv("S3_ACCESS_KEY", "s3-key")
	t.Setenv("S3_SECRET_KEY", "s3-secret")

	e, a, s := s3Settings()
	if e != "minio:9000" || a != "minio-user" || s != "minio-pass" {
		t.Fatalf("got %q %q %q", e, a, s)
	}
}
