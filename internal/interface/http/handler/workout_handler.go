package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/command"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
	"github.com/go-playground/validator/v10"
)

// WorkoutHandler handles HTTP requests for workouts.
type WorkoutHandler struct {
	createWorkoutHandler *command.CreateWorkoutHandler
	validator            *validator.Validate
}

// NewWorkoutHandler creates a new WorkoutHandler.
func NewWorkoutHandler(createWorkoutHandler *command.CreateWorkoutHandler, v *validator.Validate) *WorkoutHandler {
	return &WorkoutHandler{
		createWorkoutHandler: createWorkoutHandler,
		validator:            v,
	}
}

// ServeHTTP dispatches the request to the appropriate handler method.
func (h *WorkoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())

	switch r.Method {
	case http.MethodPost:
		if r.URL.Path == "/v1/workouts" {
			h.createWorkout(w, r)
			return
		}
	default:
		logger.Warn("Method not allowed or path not found for workout handler", "method", r.Method, "path", r.URL.Path)
		SendJSONError(w, logger, "Not Found", http.StatusNotFound, r.Method+" "+r.URL.Path+" not found")
	}
}

func (h *WorkoutHandler) createWorkout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)

	uidWithPrefix, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		logger.Error("Failed to get deviceID string from context (not found or wrong type)")
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Device ID not found in context or invalid type.")
		return
	}

	uidStr := strings.TrimPrefix(uidWithPrefix, "device:")
	if uidStr == uidWithPrefix {
		logger.Error("Device ID from context does not have expected 'device:' prefix", "rawDeviceID", uidWithPrefix)
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Invalid device ID format in token (prefix missing).")
		return
	}

	deviceID, err := entity.NewDeviceID(uidStr) // Assumes entity.NewDeviceID returns (entity.DeviceID, error)
	if err != nil {
		logger.Error("Failed to parse deviceID from context string after stripping prefix", "error", err, "parsedDeviceIDString", uidStr)
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Invalid device ID format in token: "+err.Error())
		return
	}

	var req dto.CreateWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode request body for create workout", "error", err)
		SendJSONError(w, logger, "Invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	if err := h.validator.StructCtx(ctx, req); err != nil {
		logger.Error("Validation failed for create workout request", "error", err)
		SendJSONError(w, logger, "Validation failed", http.StatusBadRequest, err.Error())
		return
	}

	cmd := command.CreateWorkoutCommand{
		DeviceID:    deviceID,
		MenuID:      req.MenuID,
		PerformedAt: req.PerformedAt,
		Notes:       req.Notes,
		Sets:        req.Sets,
	}

	result, cmdErr := h.createWorkoutHandler.Handle(ctx, cmd) // Renamed err to cmdErr to avoid scope issues
	if cmdErr != nil {
		logger.Error("Failed to create workout", "error", cmdErr, "deviceID", deviceID.String(), "menuID", cmd.MenuID)
		switch e := cmdErr.(type) {
		case *apperror.ErrNotFound:
			SendJSONError(w, logger, e.Resource+" Not Found", http.StatusNotFound, e.Error())
		case *apperror.ErrBadRequest:
			SendJSONError(w, logger, e.Message, http.StatusBadRequest, e.Details)
		case *apperror.ErrUnauthorized:
			SendJSONError(w, logger, e.Message, http.StatusUnauthorized, e.Error())
		case *apperror.ErrInternal:
			logger.Error("Internal server error during workout creation", "internal_error_message", e.Message, "cause", e.Err)
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "An unexpected error occurred.")
		default:
			logger.Error("Unhandled error type during workout creation", "error_type", fmt.Sprintf("%T", e), "error_value", e.Error())
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "An unexpected and unhandled error occurred.")
		}
		return
	}

	logger.Info("Successfully created workout", "workoutID", result.Workout.ID, "deviceID", deviceID.String())
	// Standard JSON response for success
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result.Workout); err != nil {
		logger.Error("Failed to encode successful workout response", "error", err, "workoutID", result.Workout.ID)
	}
}
