package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type invokeRequest struct {
	InvocationID string                 `json:"invocation_id"`
	RequestID    string                 `json:"request_id"`
	Request      map[string]interface{} `json:"request"`
}

type invokeResponse struct {
	Message      string `json:"message"`
	InvocationID string `json:"invocation_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	HandledAt    string `json:"handled_at"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/invoke", invokeHandler)
	mux.HandleFunc("/", defaultHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("sample-container listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	var req invokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json body",
		})
		return
	}

	resp := invokeResponse{
		Message:      "hello from edgebase sample container",
		InvocationID: req.InvocationID,
		RequestID:    req.RequestID,
		HandledAt:    time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "edgebase sample container",
		"path":    r.URL.Path,
		"method":  r.Method,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode response failed: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s host=%s duration=%s", r.Method, r.URL.Path, r.Host, time.Since(start))
	})
}
