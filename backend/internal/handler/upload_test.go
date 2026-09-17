package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/service"
)

type fakeUploader struct {
	input service.FileInput
	err   error
}

func (f *fakeUploader) Upload(_ context.Context, input service.FileInput) (service.UploadedFile, error) {
	f.input = input
	if f.err != nil {
		return service.UploadedFile{}, f.err
	}
	return service.UploadedFile{Key: "files/document.pdf", URL: "/uploads/files/document.pdf", Filename: input.Filename, Size: len(input.Data)}, nil
}

func TestUploadEndpointStoresGenericFile(t *testing.T) {
	auth := &fakeAuth{profile: service.PublicUser{ID: uuid.New(), Email: "person@example.com"}}
	uploader := &fakeUploader{}
	mux := http.NewServeMux()
	NewUploadHandler(auth, uploader).RegisterRoutes(mux)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "document.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("pdf data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/uploads", &body)
	request.Header.Set("Authorization", "Bearer access-token")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated || uploader.input.Filename != "document.pdf" || !strings.Contains(recorder.Body.String(), "/uploads/files/document.pdf") {
		t.Fatalf("upload failed: %d %s", recorder.Code, recorder.Body.String())
	}
}
