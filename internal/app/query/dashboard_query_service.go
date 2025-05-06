package query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
)

// DashboardQueryService provides methods to query aggregated dashboard data.
type DashboardQueryService struct {
	querier db.Querier // Use sqlc generated Querier interface
}

// NewDashboardQueryService creates a new DashboardQueryService.
func NewDashboardQueryService(q db.Querier) *DashboardQueryService {
	return &DashboardQueryService{
		querier: q,
	}
}

// GetWeeklyVolumeSummary retrieves the weekly volume summary for a device within a date range.
func (s *DashboardQueryService) GetWeeklyVolumeSummary(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) ([]dto.WeeklyVolumeSummaryDTO, error) {
	params := db.GetWeeklyVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   startDate.Format("2006-01-02"), // Corresponds to >= start_date
		PerformedAt_2: endDate.Format("2006-01-02"),   // Corresponds to < end_date
	}

	rows, err := s.querier.GetWeeklyVolumeSummary(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dto.WeeklyVolumeSummaryDTO{}, nil
		}
		return nil, apperror.NewErrInternal("failed to get weekly volume summary from db", err)
	}

	resultDTOs := make([]dto.WeeklyVolumeSummaryDTO, len(rows))
	for i, row := range rows {
		// Type assertion for YearWeek (interface{} -> string)
		yearWeekStr, ok := row.YearWeek.(string)
		if !ok {
			// Handle unexpected type, though strftime should return string
			return nil, apperror.NewErrInternal(fmt.Sprintf("unexpected type for year_week: %T", row.YearWeek), nil)
		}

		// Handle TotalVolume (sql.NullFloat64 -> float64, default to 0.0 if NULL)
		var totalVolume float64
		if row.TotalVolume.Valid {
			totalVolume = row.TotalVolume.Float64
		}

		resultDTOs[i] = dto.WeeklyVolumeSummaryDTO{
			YearWeek:    yearWeekStr,
			TotalVolume: totalVolume,
		}
	}

	return resultDTOs, nil
}

// GetExerciseVolumeSummary retrieves the exercise volume summary for a device within a date range.
func (s *DashboardQueryService) GetExerciseVolumeSummary(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) ([]dto.ExerciseVolumeSummaryDTO, error) {
	params := db.GetExerciseVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   startDate.Format("2006-01-02"),
		PerformedAt_2: endDate.Format("2006-01-02"),
	}

	rows, err := s.querier.GetExerciseVolumeSummary(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dto.ExerciseVolumeSummaryDTO{}, nil
		}
		return nil, apperror.NewErrInternal("failed to get exercise volume summary from db", err)
	}

	resultDTOs := make([]dto.ExerciseVolumeSummaryDTO, len(rows))
	for i, row := range rows {
		var totalVolume float64
		if row.TotalVolume.Valid {
			totalVolume = row.TotalVolume.Float64
		}
		resultDTOs[i] = dto.ExerciseVolumeSummaryDTO{
			ExerciseID:   row.ExerciseID,
			ExerciseName: row.ExerciseName,
			TotalVolume:  totalVolume,
		}
	}

	return resultDTOs, nil
}

// GetMuscleVolumeSummary retrieves the muscle volume summary for a device within a date range.
func (s *DashboardQueryService) GetMuscleVolumeSummary(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) ([]dto.MuscleVolumeSummaryDTO, error) {
	params := db.GetMuscleVolumeSummaryParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   startDate.Format("2006-01-02"),
		PerformedAt_2: endDate.Format("2006-01-02"),
	}

	rows, err := s.querier.GetMuscleVolumeSummary(ctx, params)
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
func (s *DashboardQueryService) GetAverageWeeklyVolume(ctx context.Context, deviceID entity.DeviceID, startDate, endDate time.Time) (*dto.AverageWeeklyVolumeDTO, error) {
	params := db.GetTotalVolumeAveragePerWeekParams{
		DeviceID:      deviceID.String(),
		PerformedAt:   startDate.Format("2006-01-02"),
		PerformedAt_2: endDate.Format("2006-01-02"),
	}

	avgVolumeResult, err := s.querier.GetTotalVolumeAveragePerWeek(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			// No workout data in the period, average is null.
			return &dto.AverageWeeklyVolumeDTO{AverageWeeklyVolume: nil}, nil
		}
		return nil, apperror.NewErrInternal("failed to get average weekly volume from db", err)
	}

	// avgVolumeResult is sql.NullFloat64
	var avgVolumePtr *float64
	if avgVolumeResult.Valid {
		avgVolumePtr = &avgVolumeResult.Float64
	}

	resultDTO := &dto.AverageWeeklyVolumeDTO{
		AverageWeeklyVolume: avgVolumePtr,
	}

	return resultDTO, nil
}
