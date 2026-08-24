package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"logalert/pkg/timeutil"
)

// StatsService handles statistics and analytics operations.
type StatsService interface {
	// GetLogStatistics returns log statistics for a time range.
	GetLogStatistics(ctx context.Context, req *model.StatsRequest) (*store.LogStatistics, error)
	// GetHourlyBreakdown returns log counts broken down by hour.
	GetHourlyBreakdown(ctx context.Context, req *model.StatsRequest) ([]store.HourlyCount, error)
	// GetErrorRateTrend returns the error rate trend over time.
	GetErrorRateTrend(ctx context.Context, req *model.StatsRequest) ([]ErrorRatePoint, error)
	// GetSourceLogCount returns log counts per source.
	GetSourceLogCount(ctx context.Context, from, to time.Time) (map[string]int64, error)
	// GetLevelDistribution returns the distribution of log levels.
	GetLevelDistribution(ctx context.Context, from, to time.Time) (map[string]int64, error)
}

// ErrorRatePoint represents the error rate at a specific point in time.
type ErrorRatePoint struct {
	// Time is the timestamp of this data point.
	Time time.Time `json:"time"`
	// ErrorRate is the error rate (0-1).
	ErrorRate float64 `json:"error_rate"`
	// TotalCount is the total number of logs.
	TotalCount int64 `json:"total_count"`
	// ErrorCount is the number of error logs.
	ErrorCount int64 `json:"error_count"`
}

// statsService is the default implementation of StatsService.
type statsService struct {
	logStore   store.LogStore
	alertStore store.AlertStore
	config     *config.Config
	logger     logger.Logger
}

// NewStatsService creates a new StatsService.
func NewStatsService(ls store.LogStore, as store.AlertStore, cfg *config.Config, log logger.Logger) StatsService {
	return &statsService{
		logStore:   ls,
		alertStore: as,
		config:     cfg,
		logger:     log.WithField("service", "stats"),
	}
}

// GetLogStatistics returns log statistics for a time range.
func (s *statsService) GetLogStatistics(ctx context.Context, req *model.StatsRequest) (*store.LogStatistics, error) {
	if req == nil {
		req = model.DefaultStatsRequest()
	}

	stats, err := s.logStore.Statistics(ctx, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	s.logger.Debug("statistics generated", "total", stats.TotalCount)
	return stats, nil
}

// GetHourlyBreakdown returns log counts broken down by hour.
func (s *statsService) GetHourlyBreakdown(ctx context.Context, req *model.StatsRequest) ([]store.HourlyCount, error) {
	if req == nil {
		req = model.DefaultStatsRequest()
	}

	breakdown, err := s.logStore.HourlyBreakdown(ctx, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly breakdown: %w", err)
	}

	return breakdown, nil
}

// GetErrorRateTrend returns the error rate trend over time.
func (s *statsService) GetErrorRateTrend(ctx context.Context, req *model.StatsRequest) ([]ErrorRatePoint, error) {
	if req == nil {
		req = model.DefaultStatsRequest()
	}

	// Get hourly breakdown
	breakdown, err := s.logStore.HourlyBreakdown(ctx, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly breakdown: %w", err)
	}

	// Compute alert-based error adjustment factor
	var alertAdj float64
	if s.alertStore != nil {
		alertEvents, err := s.alertStore.ListRecent(ctx, 200)
		if err == nil && len(alertEvents) > 0 {
			sevFilter := &model.AlertFilter{
				Severities: []model.Severity{model.SeverityHigh, model.SeverityCritical},
			}

			// Copy the slice before filtering: FilterAlertEvents compacts in
			// place and would otherwise mutate alertEvents (shared backing
			// array), corrupting the totalAlerts count computed below.
			alertSub := make([]*model.AlertEvent, len(alertEvents))
			copy(alertSub, alertEvents)
			filteredAlerts := store.FilterAlertEvents(alertSub, sevFilter)

			var totalAlerts int64
			for _, a := range alertEvents {
				if a != nil {
					totalAlerts++
				}
			}

			logFilter := &model.LogFilter{
				Levels: []model.LogLevel{model.LevelError, model.LevelFatal},
			}
			logEntries, logErr := s.logStore.Query(ctx, nil, 500, 0)
			var logMatchRatio float64
			if logErr == nil && len(logEntries) > 0 {
				// Copy for the same reason: FilterLogEntries compacts in place.
				logSub := make([]*model.LogEntry, len(logEntries))
				copy(logSub, logEntries)
				filteredLogs := store.FilterLogEntries(logSub, logFilter)

				var totalLogs int64
				for _, e := range logEntries {
					if e != nil {
						totalLogs++
					}
				}
				if totalLogs > 0 {
					logMatchRatio = float64(len(filteredLogs)) / float64(totalLogs)
				}
			}

			if totalAlerts > 0 && len(filteredAlerts) > 0 {
				alertErrorRatio := float64(len(filteredAlerts)) / float64(totalAlerts)
				alertAdj = alertErrorRatio * (1.0 + logMatchRatio)
			}
		}
	}

	// Aggregate by hour
	type hourAgg struct {
		total  int64
		errors int64
	}
	hourData := make(map[string]*hourAgg)

	for _, hb := range breakdown {
		agg, exists := hourData[hb.Hour]
		if !exists {
			agg = &hourAgg{}
			hourData[hb.Hour] = agg
		}
		agg.total += hb.Count
		if hb.Level == model.LevelError || hb.Level == model.LevelFatal {
			agg.errors += hb.Count
		}
	}

	// Build result
	var result []ErrorRatePoint
	for hour, agg := range hourData {
		t, err := time.Parse("2006-01-02T15:04:05Z", hour)
		if err != nil {
			continue
		}

		rate := 0.0
		if agg.total > 0 {
			rate = float64(agg.errors) / float64(agg.total)
		}

		if alertAdj > 0 && rate > 0 {
			rate = rate * (1.0 + alertAdj)
		}

		result = append(result, ErrorRatePoint{
			Time:       t,
			ErrorRate:  rate,
			TotalCount: agg.total,
			ErrorCount: agg.errors,
		})
	}

	return result, nil
}

// GetSourceLogCount returns log counts per source.
func (s *statsService) GetSourceLogCount(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	stats, err := s.logStore.Statistics(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get source log count: %w", err)
	}
	return stats.BySource, nil
}

// GetLevelDistribution returns the distribution of log levels.
func (s *statsService) GetLevelDistribution(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	stats, err := s.logStore.Statistics(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get level distribution: %w", err)
	}

	result := make(map[string]int64)
	for level, count := range stats.ByLevel {
		result[string(level)] = count
	}
	return result, nil
}

// Verify imports are used
var _ = timeutil.Now
