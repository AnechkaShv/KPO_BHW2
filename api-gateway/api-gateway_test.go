package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

var testServices map[string]ServiceConfig
var servicesMutex sync.Mutex

func init() {
	resetTestServices()
}

func resetTestServices() {
	servicesMutex.Lock()
	defer servicesMutex.Unlock()

	testServices = make(map[string]ServiceConfig)
	for k, v := range services {
		testServices[k] = v
	}
}

func TestHealthCheckHandler(t *testing.T) {
	fileStoringSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fileStoringSrv.Close()

	fileAnalysisSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fileAnalysisSrv.Close()

	resetTestServices()
	servicesMutex.Lock()
	testServices["files"] = ServiceConfig{
		Name:   "File Storing Service",
		URL:    fileStoringSrv.URL,
		Client: &http.Client{Timeout: 1 * time.Second},
	}
	testServices["analyze"] = ServiceConfig{
		Name:   "File Analysis Service",
		URL:    fileAnalysisSrv.URL,
		Client: &http.Client{Timeout: 1 * time.Second},
	}
	testServices["wordcloud"] = testServices["analyze"]
	servicesMutex.Unlock()

	origServices := services
	services = testServices
	defer func() { services = origServices }()

	t.Run("Successful health check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		healthCheckHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("Unhealthy service", func(t *testing.T) {
		badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer badSrv.Close()

		servicesMutex.Lock()
		testServices["files"] = ServiceConfig{
			Name:   "File Storing Service",
			URL:    badSrv.URL,
			Client: &http.Client{Timeout: 1 * time.Second},
		}
		servicesMutex.Unlock()

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		healthCheckHandler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}
	})
}

func TestApiHandler(t *testing.T) {
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test":"response"}`))
	}))
	defer mockSrv.Close()

	resetTestServices()
	servicesMutex.Lock()
	testServices["test"] = ServiceConfig{
		Name:   "Test Service",
		URL:    mockSrv.URL,
		Client: &http.Client{Timeout: 1 * time.Second},
	}
	servicesMutex.Unlock()

	origServices := services
	services = testServices
	defer func() { services = origServices }()

	tests := []struct {
		name           string
		url            string
		method         string
		expectedStatus int
	}{
		{
			name:           "Existing service",
			url:            "/api/test/endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existent service",
			url:            "/api/nonexistent",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rr := httptest.NewRecorder()

			apiHandler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
