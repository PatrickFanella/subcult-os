package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
)

type recordingMinioClient struct {
	called      bool
	bucket      string
	objectName  string
	contentType string
	body        string
	size        int64
	returnErr   error
}

type fakeMediaStorage struct {
	called      bool
	eventID     string
	filename    string
	contentType string
	body        string
	size        int64
	url         string
	err         error
}

func (f *fakeMediaStorage) UploadEventImage(ctx context.Context, eventID, filename, contentType string, body io.Reader, size int64) (string, error) {
	_ = ctx
	f.called = true
	f.eventID = eventID
	f.filename = filename
	f.contentType = contentType
	f.size = size
	payload, _ := io.ReadAll(body)
	f.body = string(payload)
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

func (c *recordingMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	_ = ctx
	c.called = true
	c.bucket = bucketName
	c.objectName = objectName
	c.size = objectSize
	c.contentType = opts.ContentType
	body, _ := io.ReadAll(reader)
	c.body = string(body)
	if c.returnErr != nil {
		return minio.UploadInfo{}, c.returnErr
	}
	return minio.UploadInfo{}, nil
}

func TestNewMediaStorageRejectsMissingConfig(t *testing.T) {
	if _, err := newMediaStorage(Config{}); err == nil || err.Error() != "media storage is not configured" {
		t.Fatalf("expected missing media config error, got %v", err)
	}
}

func TestNewMediaStorageRejectsInvalidEndpoint(t *testing.T) {
	_, err := newMediaStorage(Config{
		MediaS3Endpoint:    "http://[::1",
		MediaS3AccessKey:   "access",
		MediaS3SecretKey:   "secret",
		MediaS3Bucket:      "media",
		MediaPublicBaseURL: "https://media.example",
	})
	if err == nil || err.Error() != "media storage endpoint is invalid" {
		t.Fatalf("expected invalid endpoint error, got %v", err)
	}
}

func TestAppNewInitializesMediaStorageOnce(t *testing.T) {
	app := New(Config{
		AppEnv:             "test",
		PublicWebURL:       "http://example.test",
		SessionSecret:      "test-secret",
		MediaS3Endpoint:    "http://media.internal:9000",
		MediaS3AccessKey:   "access",
		MediaS3SecretKey:   "secret",
		MediaS3Bucket:      "media",
		MediaPublicBaseURL: "https://media.example",
	}, nil)
	if app.mediaErr != nil {
		t.Fatalf("expected startup media storage to initialize, got %v", app.mediaErr)
	}
	if app.media == nil {
		t.Fatal("expected media storage adapter to be injected at startup")
	}
}

func TestHandleUploadEventImageRejectsUnavailableMediaStorage(t *testing.T) {
	fx := newLifecycleFixture(t)
	fx.app.mediaErr = errors.New("media storage is not configured")

	req := httptest.NewRequest(http.MethodPost, "/api/events/test-event/image", nil)
	rec := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous media request status=%d, want 401", rec.Code)
	}
	req.AddCookie(fx.ownerCookie)
	rec = httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected service unavailable, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"error\":\"media storage is not configured\"}\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestHandleUploadEventImageUploadsAndPersistsImageURL(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Hero Event", 5)
	eventID := mustString(t, event, "id")
	media := &fakeMediaStorage{url: "https://media.example/events/" + eventID + "/hero-123.png"}
	fx.app.media = media
	fx.app.mediaErr = nil

	req := newMultipartUploadRequest(t, "/api/events/"+eventID+"/image", "image", "poster.png", "image/png", []byte("image-bytes"))
	req.AddCookie(fx.ownerCookie)
	rec := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected upload success, got %d: %s", rec.Code, rec.Body.String())
	}
	if !media.called || media.eventID != eventID || media.filename != "poster.png" || media.contentType != "image/png" || media.body != "image-bytes" {
		t.Fatalf("unexpected adapter call: %#v", media)
	}
	if !strings.Contains(rec.Body.String(), media.url) {
		t.Fatalf("expected response to include image url, got %s", rec.Body.String())
	}
}

func TestHandleUploadEventImageReturnsStableUploadError(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Hero Event", 5)
	eventID := mustString(t, event, "id")
	media := &fakeMediaStorage{err: errors.New("boom")}
	fx.app.media = media
	fx.app.mediaErr = nil

	req := newMultipartUploadRequest(t, "/api/events/"+eventID+"/image", "image", "poster.png", "image/png", []byte("image-bytes"))
	req.AddCookie(fx.ownerCookie)
	rec := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected upload failure, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"error\":\"could not upload image\"}\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	if !media.called {
		t.Fatal("expected adapter to be called")
	}
}

func TestS3MediaStorageUploadBuildsPublicURL(t *testing.T) {
	client := &recordingMinioClient{}
	storage := &s3MediaStorage{client: client, bucket: "media", publicBaseURL: "https://media.example"}

	url, err := storage.UploadEventImage(context.Background(), "abc", "hero.png", "image/png", strings.NewReader("image-bytes"), int64(len("image-bytes")))
	if err != nil {
		t.Fatalf("UploadEventImage returned error: %v", err)
	}
	if !client.called {
		t.Fatal("expected put object to be called")
	}
	if client.bucket != "media" || !strings.HasPrefix(client.objectName, "events/abc/hero-") || !strings.HasSuffix(client.objectName, ".png") {
		t.Fatalf("unexpected put target: bucket=%q object=%q", client.bucket, client.objectName)
	}
	if client.contentType != "image/png" || client.body != "image-bytes" {
		t.Fatalf("unexpected put payload: contentType=%q body=%q", client.contentType, client.body)
	}
	if url != "https://media.example/"+client.objectName {
		t.Fatalf("unexpected public url: %q", url)
	}
}

func TestS3MediaStorageUploadReturnsError(t *testing.T) {
	client := &recordingMinioClient{returnErr: errors.New("boom")}
	storage := &s3MediaStorage{client: client, bucket: "media", publicBaseURL: "https://media.example"}

	if _, err := storage.UploadEventImage(context.Background(), "abc", "hero.png", "image/png", strings.NewReader("image-bytes"), int64(len("image-bytes"))); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected upload error, got %v", err)
	}
}

func TestS3MediaStoragePublicURLCleansObjectKey(t *testing.T) {
	storage := &s3MediaStorage{publicBaseURL: "https://cdn.example.test/media"}

	got := storage.publicURL("events//image.png")
	want := "https://cdn.example.test/media/events/image.png"
	if got != want {
		t.Fatalf("publicURL() = %q, want %q", got, want)
	}
}

func TestS3MediaStoragePublicURLPreservesBaseSlashBehavior(t *testing.T) {
	storage := &s3MediaStorage{publicBaseURL: "https://cdn.example.test/media/"}

	got := storage.publicURL("events/image.png")
	want := "https://cdn.example.test/media//events/image.png"
	if got != want {
		t.Fatalf("publicURL() = %q, want %q", got, want)
	}
}

func newMultipartUploadRequest(t *testing.T, path, fieldName, filename, contentType string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
