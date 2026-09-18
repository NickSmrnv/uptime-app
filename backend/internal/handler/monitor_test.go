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
	monitors  []model.Monitor
	access    string
	monitorID uuid.UUID
	input     service.MonitorInput
	err       error
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

func (f *fakeMonitors) Update(_ context.Context, access string, monitorID uuid.UUID, input service.MonitorInput) (model.Monitor, error) {
	f.access = access
	f.monitorID = monitorID
	f.input = input
	if f.err != nil {
		return model.Monitor{}, f.err
	}
	return f.monitors[0], nil
}

func (f *fakeMonitors) Delete(_ context.Context, access string, monitorID uuid.UUID) error {
	f.access = access
	f.monitorID = monitorID
	return f.err
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

	monitorService.err = service.ErrMonitorLimitReached
	limited := httptest.NewRequest(http.MethodPost, "/monitors", strings.NewReader(`{"url":"https://example.com","intervalSeconds":60}`))
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, limited)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("limit status = %d", recorder.Code)
	}
}

func TestMonitorRoutesUpdateAndDelete(t *testing.T) {
	monitorID := uuid.New()
	monitor := model.Monitor{ID: monitorID, URL: "https://updated.example.com", IntervalSeconds: 600, CreatedAt: time.Now().UTC()}
	monitorService := &fakeMonitors{monitors: []model.Monitor{monitor}}
	mux := http.NewServeMux()
	NewMonitorHandler(monitorService).RegisterRoutes(mux)

	update := httptest.NewRequest(http.MethodPatch, "/monitors/"+monitorID.String(), strings.NewReader(`{"url":"https://updated.example.com","intervalSeconds":600}`))
	update.Header.Set("Authorization", "Bearer access-token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, update)
	if recorder.Code != http.StatusOK || monitorService.monitorID != monitorID || monitorService.input.URL != "https://updated.example.com" {
		t.Fatalf("update failed: status=%d input=%#v body=%s", recorder.Code, monitorService.input, recorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/monitors/"+monitorID.String(), nil)
	deleteRequest.Header.Set("Authorization", "Bearer access-token")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, deleteRequest)
	if recorder.Code != http.StatusNoContent || monitorService.monitorID != monitorID || monitorService.access != "access-token" {
		t.Fatalf("delete failed: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMonitorRoutesRejectInvalidIDAndNotFound(t *testing.T) {
	monitorService := &fakeMonitors{monitors: []model.Monitor{{ID: uuid.New()}}, err: service.ErrMonitorNotFound}
	mux := http.NewServeMux()
	NewMonitorHandler(monitorService).RegisterRoutes(mux)

	invalidID := httptest.NewRequest(http.MethodPatch, "/monitors/not-a-uuid", strings.NewReader(`{"url":"https://example.com","intervalSeconds":60}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, invalidID)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid update ID status = %d", recorder.Code)
	}

	notFound := httptest.NewRequest(http.MethodDelete, "/monitors/"+uuid.New().String(), nil)
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, notFound)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("not found delete status = %d", recorder.Code)
	}
}
