package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/service"
)

// stats returns weighted counts and null percentages for intervals with no observations.
// @Summary Get monitor availability history for the current URL
// @Tags monitors
// @Produce json
// @Security BearerAuth
// @Param id path string true "Monitor UUID"
// @Param period query string true "History period" Enums(1h,24h,7d,30d)
// @Success 200 {object} model.MonitorStats
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /monitors/{id}/stats [get]
func (h *MonitorHandler) stats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.writeMonitorError(w, service.ErrInvalidInput)
		return
	}
	stats, err := h.monitors.Stats(r.Context(), accessToken(r), id, r.URL.Query().Get("period"))
	if !h.writeMonitorError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
