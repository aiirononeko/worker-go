package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	sqlc "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql" // sqlc generated, aliased to sqlc
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
)

// DashboardQueryService provides methods to query dashboard-related data.
type DashboardQueryService struct {
	db sqlc.Querier // Use sqlc generated Querier interface with alias
}

// NewDashboardQueryService creates a new DashboardQueryService.
func NewDashboardQueryService(q sqlc.Querier) *DashboardQueryService {
	return &DashboardQueryService{
		db: q,
	}
}

// GetWeeklyVolumeSummary retrieves the weekly volume summary for a device within a date range.
func (qs *DashboardQueryService) GetWeeklyVolumeSummary(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) ([]dto.WeeklyVolumeSummaryDTO, error) {
	logger := middleware.LoggerFromContext(ctx)
	paramStartDateStr := startDate.Format("2006-01-02T00:00:00Z")
	paramEndDateStr := endDate.AddDate(0, 0, 1).Format("2006-01-02T00:00:00Z")

	params := sqlc.GetWeeklyVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   paramStartDateStr,
		PerformedAt_2: paramEndDateStr,
	}

	rows, err := qs.db.GetWeeklyVolumeSummary(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute GetWeeklyVolumeSummary query", "error", err, "params", params)
		if errors.Is(err, sql.ErrNoRows) { // sql.ErrNoRows is from standard library "database/sql"
			return []dto.WeeklyVolumeSummaryDTO{}, nil
		}
		return nil, apperror.NewErrInternal("Failed to get weekly volume summary from db", err)
	}

	summaries := make([]dto.WeeklyVolumeSummaryDTO, 0, len(rows))
	for _, row := range rows {
		yearWeekStr, ok := row.YearWeek.(string)
		if !ok {
			logger.WarnContext(ctx, "Unexpected type for YearWeek", "type", fmt.Sprintf("%T", row.YearWeek), "value", row.YearWeek)
			// Skip this row or handle error appropriately
			continue
		}
		summary := dto.WeeklyVolumeSummaryDTO{
			YearWeek: yearWeekStr,
		}
		if row.TotalVolume.Valid {
			summary.TotalVolume = row.TotalVolume.Float64
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// GetExerciseVolumeSummary retrieves aggregated volume data for each exercise
// performed by a device within a given date range.
func (qs *DashboardQueryService) GetExerciseVolumeSummary(
	ctx context.Context,
	deviceID entity.DeviceID,
	startDate, endDate time.Time,
) ([]dto.ExerciseVolumeSummaryDTO, error) {
	logger := middleware.LoggerFromContext(ctx)

	paramStartDateStr := startDate.Format("2006-01-02T00:00:00Z")
	paramEndDateStr := endDate.AddDate(0, 0, 1).Format("2006-01-02T00:00:00Z")

	params := sqlc.GetExerciseVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   paramStartDateStr,
		PerformedAt_2: paramEndDateStr,
	}

	rows, err := qs.db.GetExerciseVolumeSummary(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute GetExerciseVolumeSummary query", "error", err, "params", params)
		if errors.Is(err, sql.ErrNoRows) { // sql.ErrNoRows is from standard library "database/sql"
			return []dto.ExerciseVolumeSummaryDTO{}, nil
		}
		// Corrected call to NewErrInternal
		return nil, apperror.NewErrInternal("Failed to get exercise volume summary from db", err)
	}

	summaries := make([]dto.ExerciseVolumeSummaryDTO, 0, len(rows))
	for _, row := range rows {
		summary := dto.ExerciseVolumeSummaryDTO{
			ExerciseID:   row.ExerciseID,
			ExerciseName: row.ExerciseName,
			TotalSets:    row.TotalSets,
			TotalVolume:  0,
			TotalReps:    0,
			MaxWeight:    0,
		}

		if row.TotalVolume.Valid {
			summary.TotalVolume = row.TotalVolume.Float64
		}

		if row.TotalReps.Valid {
			summary.TotalReps = int64(row.TotalReps.Float64)
		}

		if row.MaxWeight != nil {
			if val, ok := row.MaxWeight.(float64); ok {
				summary.MaxWeight = val
			} else if valInt, ok := row.MaxWeight.(int64); ok {
				summary.MaxWeight = float64(valInt)
			} else {
				logger.WarnContext(ctx, "Unexpected type for MaxWeight", "type", fmt.Sprintf("%T", row.MaxWeight), "value", row.MaxWeight)
			}
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetMuscleVolumeSummary retrieves the muscle volume summary for a device within a date range.
func (s *DashboardQueryService) GetMuscleVolumeSummary(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) ([]dto.MuscleVolumeSummaryDTO, error) {
	params := sqlc.GetMuscleVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   startDate.Format("2006-01-02"),
		PerformedAt_2: endDate.Format("2006-01-02"),
	}

	rows, err := s.db.GetMuscleVolumeSummary(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dto.MuscleVolumeSummaryDTO{}, nil
		}
		return nil, apperror.NewErrInternal("failed to get muscle volume summary from db", err)
	}

	resultDTOs := make([]dto.MuscleVolumeSummaryDTO, len(rows))
	for i, row := range rows {
		var totalVolume float64
		if row.TotalVolume.Valid {
			totalVolume = row.TotalVolume.Float64
		}
		resultDTOs[i] = dto.MuscleVolumeSummaryDTO{
			MuscleID:    row.MuscleID,
			MuscleName:  row.MuscleName,
			TotalVolume: totalVolume,
		}
	}

	return resultDTOs, nil
}

// GetAverageWeeklyVolume retrieves the average weekly volume for a device within a date range.
func (qs *DashboardQueryService) GetAverageWeeklyVolume(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) (*dto.AverageWeeklyVolumeDTO, error) {
	logger := middleware.LoggerFromContext(ctx)
	paramStartDateStr := startDate.Format("2006-01-02T00:00:00Z")
	paramEndDateStr := endDate.AddDate(0, 0, 1).Format("2006-01-02T00:00:00Z")

	params := sqlc.GetTotalVolumeAveragePerWeekParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   paramStartDateStr,
		PerformedAt_2: paramEndDateStr,
	}

	average, err := qs.db.GetTotalVolumeAveragePerWeek(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // sql.ErrNoRows is from standard library "database/sql"
			// No data means average is effectively nil or 0, depending on interpretation.
			// DTO uses *float64, so return nil for the pointer.
			return &dto.AverageWeeklyVolumeDTO{AverageWeeklyVolume: nil}, nil
		}
		logger.ErrorContext(ctx, "Failed to get total volume average per week", "error", err, "params", params)
		return nil, apperror.NewErrInternal("Failed to get average weekly volume from db", err)
	}

	var avgVolumePtr *float64
	if average.Valid {
		avgVolumePtr = &average.Float64
	}

	return &dto.AverageWeeklyVolumeDTO{AverageWeeklyVolume: avgVolumePtr}, nil
}
