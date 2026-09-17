package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

const maxGenericUploadSize = 10 << 20

type FileStore interface {
	Save(context.Context, string, []byte, string) (string, error)
	Delete(context.Context, string) error
}

type FileInput struct {
	Data     []byte
	Filename string
}

type UploadedFile struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}

type UploadService struct {
	files FileStore
}

// NewUploadService binds the upload use case to a storage interface so validation can be tested
// independently of the local filesystem implementation.
func NewUploadService(files FileStore) *UploadService { return &UploadService{files: files} }

// Upload derives a basename and extension before storage to prevent client-controlled path traversal.
func (s *UploadService) Upload(ctx context.Context, input FileInput) (UploadedFile, error) {
	if s.files == nil || len(input.Data) == 0 || len(input.Data) > maxGenericUploadSize {
		return UploadedFile{}, ErrInvalidInput
	}
	filename := filepath.Base(strings.TrimSpace(input.Filename))
	if filename == "." || filename == "" || len(filename) > 255 {
		return UploadedFile{}, ErrInvalidInput
	}
	extension := strings.ToLower(filepath.Ext(filename))
	if extension == "" {
		extension = ".bin"
	}
	key, err := s.files.Save(ctx, "files", input.Data, extension)
	if err != nil {
		return UploadedFile{}, fmt.Errorf("save upload: %w", err)
	}
	return UploadedFile{Key: key, URL: "/uploads/" + key, Filename: filename, Size: len(input.Data)}, nil
}
