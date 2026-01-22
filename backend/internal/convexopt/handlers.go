package convexopt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lutexplorer/internal/common"
	"lutexplorer/internal/lut"
	"lutexplorer/internal/ws"
)

// Handlers provides HTTP handlers for the Convex Optimizer proxy API.
type Handlers struct {
	loader       *lut.Loader
	wsHub        *ws.Hub
	convexClient *Client
	disabled     bool // TODO: temporarily disabled until full module is implemented
}

// NewHandlers creates new Convex Optimizer HTTP handlers.
func NewHandlers(loader *lut.Loader, wsHub *ws.Hub, convexURL string) *Handlers {
	return &Handlers{
		loader:       loader,
		wsHub:        wsHub,
		convexClient: NewClient(convexURL),
		disabled:     true, // TODO: set to false when Convex service is ready
	}
}

// handleDisabled returns a "coming soon" response if the module is disabled.
func (h *Handlers) handleDisabled(w http.ResponseWriter) bool {
	if h.disabled {
		common.WriteError(w, http.StatusServiceUnavailable,
			"Convex optimizer is temporarily disabled. This feature will be available in a future update.")
		return true
	}
	return false
}

// HandleOptimize proxies optimization requests to the Python service.
// POST /api/convexopt/optimize
func (h *Handlers) HandleOptimize(w http.ResponseWriter, r *http.Request) {
	if h.handleDisabled(w) {
		return
	}
	if r.Method != http.MethodPost {
		common.WriteError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req ConvexOptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err.Error()))
		return
	}

	// Enrich request with file paths from loader if not absolute
	baseDir := h.loader.BaseDir()
	if !filepath.IsAbs(req.LookupFile) {
		req.LookupFile = filepath.Join(baseDir, req.LookupFile)
	}
	if !filepath.IsAbs(req.SegmentedFile) {
		req.SegmentedFile = filepath.Join(baseDir, req.SegmentedFile)
	}

	// Validate mode exists
	table, err := h.loader.GetMode(req.Mode)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, fmt.Sprintf("mode not found: %s", req.Mode))
		return
	}

	// Set cost from table if not specified
	if req.Cost <= 0 {
		req.Cost = table.Cost
		if req.Cost <= 0 {
			req.Cost = 1.0
		}
	}

	// Send to Python service
	result, err := h.convexClient.Optimize(&req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("optimization failed: %s", err.Error()))
		return
	}

	common.WriteSuccess(w, result)
}

// HandleModeInfo returns mode information for the frontend.
// GET /api/convexopt/{mode}/info
func (h *Handlers) HandleModeInfo(w http.ResponseWriter, r *http.Request) {
	if h.handleDisabled(w) {
		return
	}
	if r.Method != http.MethodGet {
		common.WriteError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	mode := extractModeFromPath(r.URL.Path, "info")
	if mode == "" {
		common.WriteError(w, http.StatusBadRequest, "mode required")
		return
	}

	// Get table
	table, err := h.loader.GetMode(mode)
	if err != nil {
		common.WriteError(w, http.StatusNotFound, fmt.Sprintf("mode not found: %s", mode))
		return
	}

	// Get mode config for file paths
	config, err := h.loader.GetModeConfig(mode)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get mode config: %s", err.Error()))
		return
	}

	// Extract criteria names from segmented file if it exists
	criteriaNames := []string{}
	segmentedFile := h.findSegmentedFile(mode)

	if segmentedFile != "" {
		if _, err := os.Stat(segmentedFile); err == nil {
			criteriaNames, _ = extractCriteriaFromFile(segmentedFile)
		}
	}

	cost := table.Cost
	if cost <= 0 {
		cost = 1.0
	}

	// Get relative path for segmented file
	segmentedRelPath := ""
	if segmentedFile != "" {
		segmentedRelPath = segmentedFile
	}

	response := ModeInfoResponse{
		Mode:          mode,
		Cost:          cost,
		CriteriaNames: criteriaNames,
		LookupFile:    config.Weights,
		SegmentedFile: segmentedRelPath,
		IsBonusMode:   cost > 1.5,
	}

	common.WriteSuccess(w, response)
}

// HandleHealth checks if the Python service is available.
// GET /api/convexopt/health
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.WriteError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	// Return disabled status if module is disabled
	if h.disabled {
		common.WriteSuccess(w, map[string]interface{}{
			"status":   "disabled",
			"message":  "Convex optimizer is temporarily disabled. This feature will be available in a future update.",
			"disabled": true,
		})
		return
	}

	health, err := h.convexClient.Health()
	if err != nil {
		common.WriteError(w, http.StatusServiceUnavailable,
			fmt.Sprintf("Convex optimizer service unavailable: %s", err.Error()))
		return
	}

	common.WriteSuccess(w, health)
}

// HandleValidate validates the configuration without running optimization.
// POST /api/convexopt/validate
func (h *Handlers) HandleValidate(w http.ResponseWriter, r *http.Request) {
	if h.handleDisabled(w) {
		return
	}
	if r.Method != http.MethodPost {
		common.WriteError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req ConvexOptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err.Error()))
		return
	}

	// Enrich request with file paths
	baseDir := h.loader.BaseDir()
	if !filepath.IsAbs(req.LookupFile) {
		req.LookupFile = filepath.Join(baseDir, req.LookupFile)
	}
	if !filepath.IsAbs(req.SegmentedFile) {
		req.SegmentedFile = filepath.Join(baseDir, req.SegmentedFile)
	}

	valid, errors, err := h.convexClient.Validate(&req)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("validation failed: %s", err.Error()))
		return
	}

	common.WriteSuccess(w, map[string]interface{}{
		"valid":  valid,
		"errors": errors,
	})
}

// RegisterRoutes registers all convex optimizer routes.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/convexopt/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/api/convexopt/health":
			h.HandleHealth(w, r)
		case path == "/api/convexopt/optimize":
			h.HandleOptimize(w, r)
		case path == "/api/convexopt/validate":
			h.HandleValidate(w, r)
		case strings.HasSuffix(path, "/info"):
			h.HandleModeInfo(w, r)
		default:
			common.WriteError(w, http.StatusNotFound, "endpoint not found")
		}
	})
}

// extractModeFromPath extracts the mode name from a URL path.
func extractModeFromPath(path, action string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	convexoptIdx := -1
	for i, p := range parts {
		if p == "convexopt" {
			convexoptIdx = i
			break
		}
	}

	if convexoptIdx < 0 || convexoptIdx+1 >= len(parts) {
		return ""
	}

	mode := parts[convexoptIdx+1]

	// Skip if mode is actually an action
	if mode == action || mode == "health" || mode == "optimize" || mode == "validate" {
		return ""
	}

	return mode
}

// findSegmentedFile finds the segmented LUT file for a mode.
// It searches in library/lookup_tables/ for files like lookUpTableSegmented_base.csv
func (h *Handlers) findSegmentedFile(mode string) string {
	libraryDir := h.loader.LibraryDir()
	if libraryDir == "" {
		// Fallback: try to find in baseDir (old behavior)
		config, err := h.loader.GetModeConfig(mode)
		if err != nil {
			return ""
		}
		segmentedFile := filepath.Join(h.loader.BaseDir(), config.Weights)
		segmentedFile = strings.Replace(segmentedFile, "lookUpTable_", "lookUpTableSegmented_", 1)
		if _, err := os.Stat(segmentedFile); err == nil {
			return segmentedFile
		}
		return ""
	}

	lookupTablesDir := filepath.Join(libraryDir, "lookup_tables")

	// Try different naming patterns
	patterns := []string{
		fmt.Sprintf("lookUpTableSegmented_%s.csv", mode),     // lookUpTableSegmented_base.csv
		fmt.Sprintf("lookUpTableSegmented_%s_0.csv", mode),   // lookUpTableSegmented_base_0.csv
		fmt.Sprintf("lookUpTableSegmented_%s", mode),         // lookUpTableSegmented_base (no extension)
		fmt.Sprintf("lookUpTableSegmented_%s_0", mode),       // lookUpTableSegmented_base_0 (no extension)
	}

	for _, pattern := range patterns {
		path := filepath.Join(lookupTablesDir, pattern)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// extractCriteriaFromFile extracts unique criteria names from a segmented LUT file.
func extractCriteriaFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	criteriaSet := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) >= 2 {
			criteria := strings.TrimSpace(parts[1])
			if criteria != "" && criteria != "0" {
				criteriaSet[criteria] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	criteria := make([]string, 0, len(criteriaSet))
	for c := range criteriaSet {
		criteria = append(criteria, c)
	}

	return criteria, nil
}
