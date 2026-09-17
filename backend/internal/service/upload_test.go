package service

import (
	"context"
	"testing"
)

type memoryFiles struct {
	category  string
	extension string
	data      []byte
}

func (m *memoryFiles) Save(_ context.Context, category string, data []byte, extension string) (string, error) {
	m.category = category
	m.extension = extension
	m.data = data
	return "files/upload.pdf", nil
}
func (m *memoryFiles) Delete(_ context.Context, _ string) error { return nil }

func TestUploadServiceStoresAnyFileInFilesCategory(t *testing.T) {
	files := &memoryFiles{}
	uploads := NewUploadService(files)
	result, err := uploads.Upload(context.Background(), FileInput{Data: []byte("document"), Filename: "report.PDF"})
	if err != nil {
		t.Fatal(err)
	}
	if files.category != "files" || files.extension != ".pdf" || result.URL != "/uploads/files/upload.pdf" || result.Filename != "report.PDF" {
		t.Fatalf("unexpected upload result: %#v, store: %#v", result, files)
	}
}
