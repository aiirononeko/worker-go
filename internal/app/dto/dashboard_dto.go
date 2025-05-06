package dto

// WeeklyVolumeSummaryDTO represents the total volume for a specific week.
type WeeklyVolumeSummaryDTO struct {
	YearWeek    string  `json:"yearWeek"` // Format: YYYY-WW (e.g., "2024-01")
	TotalVolume float64 `json:"totalVolume"`
}

// ExerciseVolumeSummaryDTO represents the total volume for a specific exercise.
type ExerciseVolumeSummaryDTO struct {
	ExerciseID   string  `json:"exerciseId"`
	ExerciseName string  `json:"exerciseName"`
	TotalVolume  float64 `json:"totalVolume"`
}

// MuscleVolumeSummaryDTO represents the total volume for a specific muscle group.
type MuscleVolumeSummaryDTO struct {
	MuscleID    string  `json:"muscleId"`
	MuscleName  string  `json:"muscleName"`
	TotalVolume float64 `json:"totalVolume"`
}

// AverageWeeklyVolumeDTO represents the average weekly volume over a period.
type AverageWeeklyVolumeDTO struct {
	// Use float64 pointer to distinguish between 0 average and no data/null average.
	AverageWeeklyVolume *float64 `json:"averageWeeklyVolume"`
}

// DashboardSummaryDTO combines key metrics for the dashboard summary.
type DashboardSummaryDTO struct {
	WeeklySummary       []WeeklyVolumeSummaryDTO `json:"weeklySummary"`
	AverageWeeklyVolume *float64                 `json:"averageWeeklyVolume"` // Directly include the average
}
