package handler

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/uptime-app/backend/internal/service"
)

const maxGenericUploadSize = 10 << 20

type ProfileReader interface {
	Profile(context.Context, string) (service.PublicUser, error)
}

type FileUploader interface {
	Upload(context.Context, service.FileInput) (service.UploadedFile, error)
}

type UploadHandler struct {
	auth    ProfileReader
	uploads FileUploader
}

func NewUploadHandler(auth ProfileReader, uploads FileUploader) *UploadHandler {
	return &UploadHandler{auth: auth, uploads: uploads}
}

func (h *UploadHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /uploads", h.upload)
}

func (h *UploadHandler) upload(w http.ResponseWriter, r *http.Request) {
	if _, err := h.auth.Profile(r.Context(), accessToken(r)); err != nil {
		writeUploadAuthError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxGenericUploadSize+(1<<20))
	defer r.Body.Close()
	if err := r.ParseMultipartForm(maxGenericUploadSize); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "file must not exceed 10 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxGenericUploadSize+1))
	if err != nil || len(data) == 0 {
		writeError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	if len(data) > maxGenericUploadSize {
		writeError(w, http.StatusRequestEntityTooLarge, "file must not exceed 10 MB")
		return
	}
	result, err := h.uploads.Upload(r.Context(), service.FileInput{Data: data, Filename: header.Filename})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid upload")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeUploadAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
