package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/service"
)

type MonitorService interface {
	Create(context.Context, string, service.MonitorInput) (model.Monitor, error)
	List(context.Context, string) ([]model.Monitor, error)
}

type MonitorHandler struct{ monitors MonitorService }

func NewMonitorHandler(monitors MonitorService) *MonitorHandler {
	return &MonitorHandler{monitors: monitors}
}

func (h *MonitorHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /monitors", h.list)
	mux.HandleFunc("POST /monitors", h.create)
}

type createMonitorRequest struct {
	URL             string `json:"url"`
	IntervalSeconds int    `json:"intervalSeconds"`
}

type monitorResponse struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"intervalSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (h *MonitorHandler) list(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.monitors.List(r.Context(), accessToken(r))
	if !h.writeMonitorError(w, err) {
		return
	}
	responses := make([]monitorResponse, 0, len(monitors))
	for _, monitor := range monitors {
		responses = append(responses, monitorFrom(monitor))
	}
	writeJSON(w, http.StatusOK, responses)
}

func (h *MonitorHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var request createMonitorRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	monitor, err := h.monitors.Create(r.Context(), accessToken(r), service.MonitorInput{URL: request.URL, IntervalSeconds: request.IntervalSeconds})
	if !h.writeMonitorError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, monitorFrom(monitor))
}

func (h *MonitorHandler) writeMonitorError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid monitor")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, service.ErrMonitorLimitReached):
		writeError(w, http.StatusUnprocessableEntity, "monitor limit reached")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return false
}

func monitorFrom(monitor model.Monitor) monitorResponse {
	return monitorResponse{ID: monitor.ID.String(), URL: monitor.URL, IntervalSeconds: monitor.IntervalSeconds, CreatedAt: monitor.CreatedAt}
}
