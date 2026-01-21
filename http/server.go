package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/textolytics/nbgo/conf"
	"github.com/textolytics/nbgo/core"
	"github.com/textolytics/nbgo/logs"
)

// Server provides HTTP API endpoints for NBGO
type Server struct {
	server          *http.Server
	coreRegistry    *core.Registry
	configManager   *conf.Manager
	logger          logs.Logger
	shutdownTimeout time.Duration
}

// NewServer creates a new HTTP API server
func NewServer(
	coreRegistry *core.Registry,
	configManager *conf.Manager,
	logger logs.Logger,
	port string,
	shutdownTimeout time.Duration,
) *Server {
	a := &Server{
		coreRegistry:    coreRegistry,
		configManager:   configManager,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", a.handleHealth)

	// API v1 endpoints
	mux.HandleFunc("/api/v1/providers", a.handleListProviders)
	mux.HandleFunc("/api/v1/status", a.handleStatus)

	a.server = &http.Server{
		Addr:           ":" + port,
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return a
}

// Start starts the HTTP server
func (a *Server) Start(ctx context.Context) error {
	a.logger.Info("Starting HTTP API server", "addr", a.server.Addr)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err.Error())
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (a *Server) Stop(ctx context.Context) error {
	a.logger.Info("Stopping HTTP API server")

	shutdownCtx, cancel := context.WithTimeout(ctx, a.shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("HTTP server shutdown error", "error", err.Error())
		return err
	}

	return nil
}

// Health check response
type HealthResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// handleHealth handles the health check endpoint
func (a *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status:  "healthy",
		Message: "NBGO system is running",
		Time:    time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Error("Failed to encode health response", "error", err.Error())
	}
}

// ProviderInfo represents provider information
type ProviderInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ProvidersResponse represents the response for the providers endpoint
type ProvidersResponse struct {
	Status    string         `json:"status"`
	Count     int            `json:"count"`
	Providers []ProviderInfo `json:"providers"`
}

// handleListProviders handles the list providers endpoint
func (a *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	providers := a.coreRegistry.List()
	providerInfos := make([]ProviderInfo, 0, len(providers))

	for _, provider := range providers {
		providerInfos = append(providerInfos, ProviderInfo{
			Name:        provider.GetName(),
			Type:        "core",
			Description: fmt.Sprintf("%s provider", provider.GetName()),
		})
	}

	response := ProvidersResponse{
		Status:    "success",
		Count:     len(providerInfos),
		Providers: providerInfos,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Error("Failed to encode providers response", "error", err.Error())
	}
}

// StatusResponse represents the status response
type StatusResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    int64                  `json:"uptime_seconds"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// handleStatus handles the status endpoint
func (a *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	config := a.configManager.Get()
	configMap := map[string]interface{}{
		"version": config.Version,
	}

	response := StatusResponse{
		Status:    "ok",
		Version:   config.Version,
		Timestamp: time.Now(),
		Uptime:    0, // Would be calculated from start time
		Config:    configMap,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Error("Failed to encode status response", "error", err.Error())
	}
}
