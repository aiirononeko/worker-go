package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
)

const dateLayout = "2006-01-02" // YYYY-MM-DD format

// DashboardHandler handles HTTP requests for dashboard data.
type DashboardHandler struct {
	queryService *query.DashboardQueryService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(qs *query.DashboardQueryService) *DashboardHandler {
	return &DashboardHandler{
		queryService: qs,
	}
}

// ServeHTTP dispatches the request to the appropriate handler method based on path.
func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		switch r.URL.Path {
		case "/v1/dashboard/summary":
			h.getSummary(w, r)
			return
		case "/v1/dashboard/exercises": // Added route for exercises
			h.getExerciseSummary(w, r)
			return
		// Add cases for other dashboard endpoints like /v1/dashboard/muscles
		default:
			logger.Warn("Path not found for GET request in dashboard handler", "path", r.URL.Path)
			SendJSONError(w, logger, "Not Found", http.StatusNotFound, "Path "+r.URL.Path+" not found")
		}
	default:
		logger.Warn("Method not allowed for dashboard handler", "method", r.Method, "path", r.URL.Path)
		SendJSONError(w, logger, "Method Not Allowed", http.StatusMethodNotAllowed, "Method "+r.Method+" not allowed for "+r.URL.Path)
	}
}

// getSummary handles requests for the dashboard summary.
// It expects 'startDate' and 'endDate' query parameters in YYYY-MM-DD format.
func (h *DashboardHandler) getSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)

	uidWithPrefix, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Device ID not found in context or invalid type.")
		return
	}
	uidStr := strings.TrimPrefix(uidWithPrefix, "device:")
	deviceID, err := entity.NewDeviceID(uidStr)
	if err != nil {
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Invalid device ID format in token: "+err.Error())
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	if startDateStr == "" || endDateStr == "" {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, "Missing required query parameters: startDate, endDate (YYYY-MM-DD)")
		return
	}

	startDate, err := time.Parse(dateLayout, startDateStr)
	if err != nil {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid startDate format: %s. Use YYYY-MM-DD.", err.Error()))
		return
	}
	endDate, err := time.Parse(dateLayout, endDateStr)
	if err != nil {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid endDate format: %s. Use YYYY-MM-DD.", err.Error()))
		return
	}

	// Fetch data using the query service
	weeklySummary, err := h.queryService.GetWeeklyVolumeSummary(ctx, deviceID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get weekly volume summary", "error", err, "deviceID", deviceID.String())
		SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve weekly volume data.")
		return
	}

	averageWeeklyVolumeDTO, err := h.queryService.GetAverageWeeklyVolume(ctx, deviceID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get average weekly volume", "error", err, "deviceID", deviceID.String())
		SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve average weekly volume data.")
		return
	}

	// Combine results into the summary DTO
	summaryDTO := dto.DashboardSummaryDTO{
		WeeklySummary:       weeklySummary,
		AverageWeeklyVolume: averageWeeklyVolumeDTO.AverageWeeklyVolume,
	}

	// Send successful response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(summaryDTO); err != nil {
		logger.Error("Failed to encode dashboard summary response", "error", err)
	}
}

// getExerciseSummary handles requests for exercise volume summary.
// It expects 'startDate' and 'endDate' query parameters in YYYY-MM-DD format.
func (h *DashboardHandler) getExerciseSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)

	uidWithPrefix, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Device ID not found in context or invalid type.")
		return
	}
	uidStr := strings.TrimPrefix(uidWithPrefix, "device:")
	deviceID, err := entity.NewDeviceID(uidStr)
	if err != nil {
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Invalid device ID format in token: "+err.Error())
		return
	}

	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")
	if startDateStr == "" || endDateStr == "" {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, "Missing required query parameters: startDate, endDate (YYYY-MM-DD)")
		return
	}
	startDate, err := time.Parse(dateLayout, startDateStr)
	if err != nil {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid startDate format: %s. Use YYYY-MM-DD.", err.Error()))
		return
	}
	endDate, err := time.Parse(dateLayout, endDateStr)
	if err != nil {
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid endDate format: %s. Use YYYY-MM-DD.", err.Error()))
		return
	}

	exerciseSummary, err := h.queryService.GetExerciseVolumeSummary(ctx, deviceID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get exercise volume summary", "error", err, "deviceID", deviceID.String())
		SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve exercise volume data.")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(exerciseSummary); err != nil {
		logger.Error("Failed to encode exercise summary response", "error", err)
	}
}

// TODO: Implement getMuscleSummary handler
