package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAPICORSHandlesCredentialedPreflight(t *testing.T) {
	handler := WithAPICORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("origin header is missing")
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials header is missing")
	}
}

func TestWithAPICORSRejectsUnknownOrigin(t *testing.T) {
	handler := WithAPICORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unexpected CORS origin header")
	}
}

func TestWithAPICORSAllowsProfileRequest(t *testing.T) {
	handler := WithAPICORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPatch, "/users/me", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("profile CORS origin header is missing")
	}
}

func TestWithAPICORSAllowsGenericUpload(t *testing.T) {
	handler := WithAPICORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("upload CORS origin header is missing")
	}
}
