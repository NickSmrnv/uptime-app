package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	uploadCategory  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	uploadFilename  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[a-z0-9]{1,16}$`)
	uploadExtension = regexp.MustCompile(`^\.[a-z0-9]{1,16}$`)
)

type UploadStorage struct {
	directory string
}

func NewUploadStorage(directory string) (*UploadStorage, error) {
	if directory == "" {
		return nil, errors.New("upload directory is empty")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &UploadStorage{directory: directory}, nil
}

func (s *UploadStorage) Save(ctx context.Context, category string, data []byte, extension string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !uploadCategory.MatchString(category) {
		return "", errors.New("invalid upload category")
	}
	extension = strings.ToLower(extension)
	if !uploadExtension.MatchString(extension) {
		return "", errors.New("invalid upload extension")
	}
	directory := filepath.Join(s.directory, category)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create upload category: %w", err)
	}
	filename := uuid.NewString() + extension
	temporary, err := os.CreateTemp(directory, ".upload-*")
	if err != nil {
		return "", fmt.Errorf("create upload: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("set upload permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("write upload: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close upload: %w", err)
	}
	key := filepath.ToSlash(filepath.Join(category, filename))
	if err := os.Rename(temporaryName, filepath.Join(s.directory, key)); err != nil {
		return "", fmt.Errorf("store upload: %w", err)
	}
	return key, nil
}

func (s *UploadStorage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validUploadKey(key) {
		return errors.New("invalid upload key")
	}
	err := os.Remove(filepath.Join(s.directory, filepath.FromSlash(key)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete upload: %w", err)
	}
	return nil
}

func (s *UploadStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if !validUploadKey(key) {
		http.NotFound(w, r)
		return
	}
	category, filename, _ := strings.Cut(key, "/")
	if category != "avatars" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	http.ServeFile(w, r, filepath.Join(s.directory, filepath.FromSlash(key)))
}

func validUploadKey(key string) bool {
	category, filename, ok := strings.Cut(key, "/")
	return ok && !strings.Contains(filename, "/") && uploadCategory.MatchString(category) && uploadFilename.MatchString(filename)
}
