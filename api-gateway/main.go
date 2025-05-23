package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ServiceConfig struct {
	Name   string
	URL    string
	Client *http.Client
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

var (
	services = map[string]ServiceConfig{
		"files": {
			Name:   "File Storing Service",
			URL:    getEnv("FILE_STORING_SERVICE_URL", "http://file-storing-service:8081"),
			Client: &http.Client{Timeout: 10 * time.Second},
		},
		"analyze": {
			Name:   "File Analysis Service",
			URL:    getEnv("FILE_ANALYSIS_SERVICE_URL", "http://file-analysis-service:8082"),
			Client: &http.Client{Timeout: 15 * time.Second},
		},
		"wordcloud": {
			Name:   "Word Cloud Service",
			URL:    getEnv("FILE_ANALYSIS_SERVICE_URL", "http://word-cloud-service:8083"),
			Client: &http.Client{Timeout: 15 * time.Second},
		},
	}
)

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	http.HandleFunc("/api/", apiHandler)
	http.HandleFunc("/health", healthCheckHandler)

	log.Println("API Gateway is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	serviceName := parts[0]

	service, exists := services[serviceName]
	if !exists {
		sendError(w, fmt.Sprintf("Service '%s' not found", serviceName), http.StatusNotFound)
		return
	}
	if len(parts) > 0 {
		serviceName += "/"
	}

	// Prepare request to backend service
	targetURL := service.URL + "/" + serviceName + strings.Join(parts[1:], "/")
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		sendError(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Add X-Forwarded headers
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-Proto", "http")

	log.Printf("Forwarding request to %s: %s %s", service.Name, r.Method, targetURL)

	// Send request to backend service
	resp, err := service.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			sendError(w, fmt.Sprintf("%s timeout", service.Name), http.StatusGatewayTimeout)
		} else {
			sendError(w, fmt.Sprintf("%s unavailable", service.Name), http.StatusBadGateway)
		}
		log.Printf("Service error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Failed to write response: %v", err)
	}

	log.Printf("Completed %s %s in %v (status: %d)",
		r.Method, r.URL.Path, time.Since(start), resp.StatusCode)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	status := make(map[string]string)
	allHealthy := true

	for name, service := range services {
		req, err := http.NewRequest("GET", service.URL+"/health", nil)
		if err != nil {
			status[name] = "error"
			allHealthy = false
			continue
		}

		resp, err := service.Client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			status[name] = "unhealthy"
			allHealthy = false
		} else {
			status[name] = "healthy"
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	response := map[string]interface{}{
		"status":   status,
		"healthy":  allHealthy,
		"datetime": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if !allHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

func sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}
