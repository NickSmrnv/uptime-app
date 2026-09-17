package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/service"
)

type fakeMonitors struct {
	monitors []model.Monitor
	access   string
	input    service.MonitorInput
	err      error
}

func (f *fakeMonitors) Create(_ context.Context, access string, input service.MonitorInput) (model.Monitor, error) {
	f.access = access
	f.input = input
	if f.err != nil {
		return model.Monitor{}, f.err
	}
	return f.monitors[0], nil
}

func (f *fakeMonitors) List(_ context.Context, access string) ([]model.Monitor, error) {
	f.access = access
	return f.monitors, f.err
}

func TestMonitorRoutesCreateAndListForAuthenticatedUser(t *testing.T) {
	monitor := model.Monitor{ID: uuid.New(), URL: "https://example.com", IntervalSeconds: 300, CreatedAt: time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)}
	monitorService := &fakeMonitors{monitors: []model.Monitor{monitor}}
	mux := http.NewServeMux()
	NewMonitorHandler(monitorService).RegisterRoutes(mux)

	create := httptest.NewRequest(http.MethodPost, "/monitors", strings.NewReader(`{"url":"https://example.com","intervalSeconds":300}`))
	create.Header.Set("Authorization", "Bearer access-token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, create)
	if recorder.Code != http.StatusCreated || monitorService.access != "access-token" || monitorService.input.IntervalSeconds != 300 || monitorService.input.URL != "https://example.com" {
		t.Fatalf("create failed: status=%d input=%#v body=%s", recorder.Code, monitorService.input, recorder.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/monitors", nil)
	list.Header.Set("Authorization", "Bearer access-token")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, list)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), monitor.ID.String()) || !strings.Contains(recorder.Body.String(), "intervalSeconds") {
		t.Fatalf("list failed: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMonitorRoutesRejectInvalidBodyAndUnauthorizedRequests(t *testing.T) {
	monitorService := &fakeMonitors{monitors: []model.Monitor{{ID: uuid.New()}}, err: errors.New("unused")}
	mux := http.NewServeMux()
	NewMonitorHandler(monitorService).RegisterRoutes(mux)

	invalid := httptest.NewRequest(http.MethodPost, "/monitors", strings.NewReader(`{"url":"https://example.com","extra":true}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, invalid)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d", recorder.Code)
	}

	monitorService.err = service.ErrUnauthorized
	unauthorized := httptest.NewRequest(http.MethodGet, "/monitors", nil)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, unauthorized)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", recorder.Code)
	}
}
