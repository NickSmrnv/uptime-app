package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadStorageSavesServesAndDeletesFile(t *testing.T) {
	storage, err := NewUploadStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, err := storage.Save(context.Background(), "documents", []byte("document data"), ".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "documents/") || !strings.HasSuffix(key, ".pdf") {
		t.Fatalf("key = %q", key)
	}
	request := httptest.NewRequest(http.MethodGet, "/uploads/"+key, nil)
	recorder := httptest.NewRecorder()
	storage.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "document data" {
		t.Fatalf("served response = %d %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Disposition") == "" {
		t.Fatal("generic upload must be served as attachment")
	}
	if err := storage.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(storage.directory, key)); !os.IsNotExist(err) {
		t.Fatalf("upload still exists: %v", err)
	}
}

func TestUploadStorageAllowsInlineAvatarAndRejectsUnsafeKeys(t *testing.T) {
	storage, err := NewUploadStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, err := storage.Save(context.Background(), "avatars", []byte("image data"), ".png")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/uploads/"+key, nil)
	recorder := httptest.NewRecorder()
	storage.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Disposition") != "" {
		t.Fatalf("avatar response = %d %#v", recorder.Code, recorder.Header())
	}
	request = httptest.NewRequest(http.MethodGet, "/uploads/../secret", nil)
	recorder = httptest.NewRecorder()
	storage.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}
