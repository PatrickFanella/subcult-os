package app

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// mediaStorage is the Adapter at the event media storage Seam.
type mediaStorage interface {
	UploadEventImage(ctx context.Context, eventID, filename, contentType string, body io.Reader, size int64) (string, error)
}

type s3MediaStorage struct {
	client        minioObjectPutter
	bucket        string
	publicBaseURL string
	privateBucket string
}

type minioObjectPutter interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

func newMediaStorage(config Config) (mediaStorage, error) {
	endpoint := strings.TrimSpace(config.MediaS3Endpoint)
	accessKey := strings.TrimSpace(config.MediaS3AccessKey)
	secretKey := strings.TrimSpace(config.MediaS3SecretKey)
	bucket := strings.TrimSpace(config.MediaS3Bucket)
	publicBaseURL := strings.TrimRight(strings.TrimSpace(config.MediaPublicBaseURL), "/")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" || publicBaseURL == "" {
		return nil, fmt.Errorf("media storage is not configured")
	}
	endpointHost, secure, err := normalizeMinIOEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("media storage endpoint is invalid")
	}
	client, err := minio.New(endpointHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Region: strings.TrimSpace(config.MediaS3Region),
	})
	if err != nil {
		return nil, fmt.Errorf("media storage endpoint is invalid")
	}
	return &s3MediaStorage{client: client, bucket: bucket, publicBaseURL: publicBaseURL, privateBucket: strings.TrimSpace(config.MediaS3PrivateBucket)}, nil
}

func (m *s3MediaStorage) UploadEventImage(ctx context.Context, eventID, filename, contentType string, body io.Reader, size int64) (string, error) {
	if m == nil || m.client == nil {
		return "", fmt.Errorf("media storage is not configured")
	}
	objectKey := fmt.Sprintf("events/%s/hero-%d%s", strings.TrimSpace(eventID), time.Now().UTC().UnixNano(), imageExtension(filename, contentType))
	if _, err := m.client.PutObject(ctx, m.bucket, objectKey, body, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", err
	}
	if url := m.publicURL(objectKey); url != "" {
		return url, nil
	}
	return "", fmt.Errorf("media public base url is not configured")
}

func (m *s3MediaStorage) publicURL(objectKey string) string {
	if m == nil || m.publicBaseURL == "" {
		return ""
	}
	return m.publicBaseURL + "/" + path.Clean(objectKey)
}

func imageExtension(filename string, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return ext
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".img"
	}
}

func normalizeMinIOEndpoint(raw string) (endpoint string, secure bool, err error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		parsed, parseErr := url.Parse(value)
		if parseErr != nil || parsed.Host == "" {
			return "", false, parseErr
		}
		return parsed.Host, parsed.Scheme == "https", nil
	}
	return value, true, nil
}

// Stage plots never use the public bucket or return an object-store URL.
// The operator provisions a separate bucket that denies anonymous access.
type privateMediaStorage interface {
	UploadStagePlot(context.Context, string, string, string, io.Reader, int64) (string, error)
	ReadStagePlot(context.Context, string) (io.ReadCloser, error)
	RemoveStagePlot(context.Context, string) error
}
type minioPrivateObjects interface {
	GetObject(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(context.Context, string, string, minio.RemoveObjectOptions) error
}

func (m *s3MediaStorage) privateReady() bool {
	return m != nil && m.client != nil && m.privateBucket != "" && m.privateBucket != m.bucket
}
func (m *s3MediaStorage) UploadStagePlot(ctx context.Context, eventID, actID, contentType string, body io.Reader, size int64) (string, error) {
	if !m.privateReady() {
		return "", fmt.Errorf("private media storage is not configured")
	}
	key := "events/" + eventID + "/tech/" + actID + "/" + uuid.NewString() + imageExtension("", contentType)
	_, err := m.client.PutObject(ctx, m.privateBucket, key, body, size, minio.PutObjectOptions{ContentType: contentType, CacheControl: "private, no-store"})
	return key, err
}
func (m *s3MediaStorage) ReadStagePlot(ctx context.Context, key string) (io.ReadCloser, error) {
	if !m.privateReady() {
		return nil, fmt.Errorf("private media storage is not configured")
	}
	client, ok := m.client.(minioPrivateObjects)
	if !ok {
		return nil, fmt.Errorf("private media reads are unavailable")
	}
	return client.GetObject(ctx, m.privateBucket, key, minio.GetObjectOptions{})
}
func (m *s3MediaStorage) RemoveStagePlot(ctx context.Context, key string) error {
	if !m.privateReady() {
		return fmt.Errorf("private media storage is not configured")
	}
	client, ok := m.client.(minioPrivateObjects)
	if !ok {
		return fmt.Errorf("private media deletion is unavailable")
	}
	return client.RemoveObject(ctx, m.privateBucket, key, minio.RemoveObjectOptions{})
}
