package documents

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const (
	Bucket          = "policy-documents"
	bucketLifeYears = 7 // ACPR archive requirement
)

// MinIOStore handles upload and presigned URL generation for policy documents.
type MinIOStore struct {
	client    *minio.Client
	urlTTL    time.Duration
	publicURL string // external URL base (empty = use endpoint)
}

// NewMinIOStore creates and initialises the MinIO client, ensuring the bucket exists.
func NewMinIOStore(ctx context.Context, endpoint, accessKey, secretKey string, useSSL bool, urlTTL time.Duration, publicURL string) (*MinIOStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	if err := ensureBucket(ctx, client); err != nil {
		return nil, err
	}

	return &MinIOStore{client: client, urlTTL: urlTTL, publicURL: publicURL}, nil
}

func ensureBucket(ctx context.Context, client *minio.Client) error {
	exists, err := client.BucketExists(ctx, Bucket)
	if err != nil {
		return fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, Bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("minio make bucket: %w", err)
		}
	}

	// 7-year retention lifecycle rule (ACPR)
	lc := lifecycle.NewConfiguration()
	lc.Rules = []lifecycle.Rule{{
		ID:     "acpr-7yr-retention",
		Status: "Enabled",
		Expiration: lifecycle.Expiration{
			Days: lifecycle.ExpirationDays(bucketLifeYears * 365),
		},
	}}
	_ = client.SetBucketLifecycle(ctx, Bucket, lc) // best-effort; existing rule is fine
	return nil
}

// Upload stores the document bytes and returns (objectKey, presignedURL, error).
func (s *MinIOStore) Upload(ctx context.Context, policyID uuid.UUID, data []byte) (string, string, error) {
	key := fmt.Sprintf("policies/%s/attestation-%s.pdf",
		policyID.String(),
		time.Now().UTC().Format("20060102150405"),
	)

	_, err := s.client.PutObject(ctx, Bucket, key,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/pdf"},
	)
	if err != nil {
		return "", "", fmt.Errorf("minio put: %w", err)
	}

	signed, err := s.presign(ctx, key)
	if err != nil {
		return key, "", err
	}
	return key, signed, nil
}

// PresignKey generates a fresh presigned URL for an existing object key.
func (s *MinIOStore) PresignKey(ctx context.Context, key string) (string, error) {
	return s.presign(ctx, key)
}

func (s *MinIOStore) presign(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, Bucket, key, s.urlTTL, url.Values{})
	if err != nil {
		return "", fmt.Errorf("minio presign: %w", err)
	}
	// Rewrite host if publicURL is configured (e.g. internal MinIO vs external ingress URL)
	if s.publicURL != "" {
		base, perr := url.Parse(s.publicURL)
		if perr == nil {
			u.Scheme = base.Scheme
			u.Host = base.Host
		}
	}
	return u.String(), nil
}
