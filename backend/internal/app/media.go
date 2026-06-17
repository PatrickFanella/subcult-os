package app

import (
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxEventImageBytes = 12 << 20

func (a *App) handleUploadEventImage(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	media, err := a.mediaStorage()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	_ = actorID

	r.Body = http.MaxBytesReader(w, r.Body, maxEventImageBytes)
	if err := r.ParseMultipartForm(maxEventImageBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid image upload")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	contentType := imageContentType(header)
	if contentType == "" {
		writeError(w, http.StatusBadRequest, "image must be jpeg, png, webp, or gif")
		return
	}
	if header.Size <= 0 || header.Size > maxEventImageBytes {
		writeError(w, http.StatusBadRequest, "image is too large")
		return
	}

	objectKey := fmt.Sprintf("events/%s/hero-%d%s", event.ID, time.Now().UTC().UnixNano(), imageExtension(header.Filename, contentType))
	imageURL, err := media.put(r.Context(), objectKey, file, header.Size, contentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not upload image")
		return
	}

	if _, err := a.db.Exec(r.Context(), `update events set image_url = $2, updated_at = now() where id = $1`, event.ID, imageURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not attach image")
		return
	}
	event.ImageURL = sql.NullString{String: imageURL, Valid: true}
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
}

type mediaStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func (a *App) mediaStorage() (mediaStorage, error) {
	endpoint := strings.TrimSpace(a.config.MediaS3Endpoint)
	accessKey := strings.TrimSpace(a.config.MediaS3AccessKey)
	secretKey := strings.TrimSpace(a.config.MediaS3SecretKey)
	bucket := strings.TrimSpace(a.config.MediaS3Bucket)
	publicBaseURL := strings.TrimRight(strings.TrimSpace(a.config.MediaPublicBaseURL), "/")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" || publicBaseURL == "" {
		return mediaStorage{}, fmt.Errorf("media storage is not configured")
	}
	endpointHost, secure, err := normalizeMinIOEndpoint(endpoint)
	if err != nil {
		return mediaStorage{}, fmt.Errorf("media storage endpoint is invalid")
	}
	client, err := minio.New(endpointHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Region: strings.TrimSpace(a.config.MediaS3Region),
	})
	if err != nil {
		return mediaStorage{}, fmt.Errorf("media storage endpoint is invalid")
	}
	return mediaStorage{client: client, bucket: bucket, publicBaseURL: publicBaseURL}, nil
}

func (m mediaStorage) put(ctx context.Context, objectKey string, file multipart.File, size int64, contentType string) (string, error) {
	if _, err := m.client.PutObject(ctx, m.bucket, objectKey, file, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", err
	}
	if m.publicBaseURL != "" {
		return m.publicBaseURL + "/" + path.Clean(objectKey), nil
	}
	return "", fmt.Errorf("media public base url is not configured")
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

func imageContentType(header *multipart.FileHeader) string {
	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return contentType
	default:
		return ""
	}
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
