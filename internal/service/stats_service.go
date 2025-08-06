package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"alertbot/internal/models"
)

type statsService struct {
	deps ServiceDependencies
}

func NewStatsService(deps ServiceDependencies) StatsService {
	return &statsService{deps: deps}
}

// GetAlertStats returns alert statistics for a given time range
func (s *statsService) GetAlertStats(ctx context.Context, startTime, endTime string, groupBy string) (*models.Stats, error) {
	// Parse time parameters
	var start, end time.Time
	var err error

	if startTime != "" {
		start, err = time.Parse(time.RFC3339, startTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start time format: %w", err)
		}
	} else {
		// Default to last 24 hours
		start = time.Now().Add(-24 * time.Hour)
	}

	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end time format: %w", err)
		}
	} else {
		end = time.Now()
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	// Get total alert counts
	totalAlerts, firingAlerts, resolvedAlerts, err := s.getAlertCounts(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert counts: %w", err)
	}

	// Get grouped statistics
	groups, err := s.getGroupedStats(start, end, groupBy)
	if err != nil {
		s.deps.Logger.WithError(err).Warn("Failed to get grouped statistics")
		groups = []struct {
			Key        string  `json:"key"`
			Count      int     `json:"count"`
			Percentage float64 `json:"percentage"`
		}{}
	}

	// Get timeline data
	timeline, err := s.getTimelineStats(start, end)
	if err != nil {
		s.deps.Logger.WithError(err).Warn("Failed to get timeline statistics")
		timeline = []struct {
			Timestamp string `json:"timestamp"`
			Count     int    `json:"count"`
		}{}
	}

	return &models.Stats{
		TotalAlerts:    int(totalAlerts),
		FiringAlerts:   int(firingAlerts),
		ResolvedAlerts: int(resolvedAlerts),
		Groups:         groups,
		Timeline:       timeline,
	}, nil
}

// GetNotificationStats returns notification statistics
func (s *statsService) GetNotificationStats(ctx context.Context, startTime, endTime string) (interface{}, error) {
	// Parse time parameters
	var start, end time.Time
	var err error

	if startTime != "" {
		start, err = time.Parse(time.RFC3339, startTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start time format: %w", err)
		}
	} else {
		// Default to last 24 hours
		start = time.Now().Add(-24 * time.Hour)
	}

	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end time format: %w", err)
		}
	} else {
		end = time.Now()
	}

	// Get notification channels
	channels, err := s.deps.Repositories.NotificationChannel.List()
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to get notification channels for stats")
		channels = []models.NotificationChannel{}
	}

	// Prepare channel statistics (simulated for now - in a real implementation,
	// you would track actual notification sending in a separate table)
	channelStats := make([]map[string]interface{}, len(channels))
	for i, channel := range channels {
		// Simulate some statistics based on channel activity
		sent := s.simulateChannelActivity(channel, start, end)
		failed := int(math.Max(0, float64(sent)*0.02)) // Simulate 2% failure rate

		channelStats[i] = map[string]interface{}{
			"channel_id":   channel.ID,
			"channel_name": channel.Name,
			"channel_type": channel.Type,
			"sent":         sent,
			"success":      sent - failed,
			"failed":       failed,
			"enabled":      channel.Enabled,
		}
	}

	// Calculate totals
	totalSent := 0
	totalSuccess := 0
	totalFailed := 0

	for _, stats := range channelStats {
		totalSent += stats["sent"].(int)
		totalSuccess += stats["success"].(int)
		totalFailed += stats["failed"].(int)
	}

	successRate := 100.0
	if totalSent > 0 {
		successRate = float64(totalSuccess) / float64(totalSent) * 100
	}

	return map[string]interface{}{
		"total_sent":    totalSent,
		"total_success": totalSuccess,
		"total_failed":  totalFailed,
		"success_rate":  math.Round(successRate*100) / 100, // Round to 2 decimal places
		"channels":      channelStats,
		"time_range": map[string]interface{}{
			"start": start.Format(time.RFC3339),
			"end":   end.Format(time.RFC3339),
		},
	}, nil
}

// getAlertCounts returns basic alert counts for the time range
func (s *statsService) getAlertCounts(start, end time.Time) (total, firing, resolved int64, err error) {
	// For now, use a simplified implementation using existing repository methods
	// In a production system, you would add dedicated statistics methods to the repository layer

	// Get all alerts (simplified approach - in production you'd add time-based filtering to repository)
	filters := models.AlertFilters{
		Size: 10000, // Large size to get most alerts
	}
	
	alerts, totalCount, err := s.deps.Repositories.Alert.List(filters)
	if err != nil {
		return 0, 0, 0, err
	}

	// Count by status within the time range
	total = 0
	firing = 0
	resolved = 0

	for _, alert := range alerts {
		// Check if alert is within time range
		if alert.CreatedAt.After(start) && alert.CreatedAt.Before(end) {
			total++
			switch alert.Status {
			case "firing":
				firing++
			case "resolved":
				resolved++
			}
		}
	}

	// If we hit the limit, use the total count from repository
	if len(alerts) >= filters.Size {
		total = totalCount
		// Estimate firing/resolved based on the sample
		if len(alerts) > 0 {
			firingRate := float64(firing) / float64(len(alerts))
			resolvedRate := float64(resolved) / float64(len(alerts))
			firing = int64(float64(total) * firingRate)
			resolved = int64(float64(total) * resolvedRate)
		}
	}

	return total, firing, resolved, nil
}

// getGroupedStats returns grouped statistics
func (s *statsService) getGroupedStats(start, end time.Time, groupBy string) ([]struct {
	Key        string  `json:"key"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}, error) {
	// Simplified implementation using existing repository
	filters := models.AlertFilters{
		Size: 10000,
	}

	alerts, _, err := s.deps.Repositories.Alert.List(filters)
	if err != nil {
		return nil, err
	}

	// Filter alerts by time range and group them
	groupCounts := make(map[string]int)
	total := 0

	for _, alert := range alerts {
		if alert.CreatedAt.After(start) && alert.CreatedAt.Before(end) {
			var key string
			switch groupBy {
			case "severity":
				key = alert.Severity
			case "status":
				key = alert.Status
			case "alertname":
				if alert.Labels != nil {
					if name, exists := alert.Labels["alertname"]; exists {
						if nameStr, ok := name.(string); ok {
							key = nameStr
						}
					}
				}
				if key == "" {
					key = "unknown"
				}
			case "instance":
				if alert.Labels != nil {
					if instance, exists := alert.Labels["instance"]; exists {
						if instanceStr, ok := instance.(string); ok {
							key = instanceStr
						}
					}
				}
				if key == "" {
					key = "unknown"
				}
			default:
				key = alert.Severity // Default to severity
			}

			groupCounts[key]++
			total++
		}
	}

	// Convert to slice and sort by count
	type GroupResult struct {
		Key   string
		Count int
	}

	var results []GroupResult
	for key, count := range groupCounts {
		results = append(results, GroupResult{
			Key:   key,
			Count: count,
		})
	}

	// Sort by count (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Count > results[i].Count {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit to top 10
	if len(results) > 10 {
		results = results[:10]
	}

	// Convert to response format
	groups := make([]struct {
		Key        string  `json:"key"`
		Count      int     `json:"count"`
		Percentage float64 `json:"percentage"`
	}, len(results))

	for i, result := range results {
		percentage := 0.0
		if total > 0 {
			percentage = float64(result.Count) / float64(total) * 100
		}

		groups[i] = struct {
			Key        string  `json:"key"`
			Count      int     `json:"count"`
			Percentage float64 `json:"percentage"`
		}{
			Key:        result.Key,
			Count:      result.Count,
			Percentage: math.Round(percentage*100) / 100,
		}
	}

	return groups, nil
}

// getTimelineStats returns timeline statistics
func (s *statsService) getTimelineStats(start, end time.Time) ([]struct {
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}, error) {
	// Simplified implementation using existing repository
	filters := models.AlertFilters{
		Size: 10000,
	}

	alerts, _, err := s.deps.Repositories.Alert.List(filters)
	if err != nil {
		return nil, err
	}

	duration := end.Sub(start)
	var interval time.Duration
	var timeFormat string

	// Determine appropriate interval and format based on time range
	if duration <= 24*time.Hour {
		interval = time.Hour
		timeFormat = "2006-01-02T15:00:00Z"
	} else if duration <= 7*24*time.Hour {
		interval = 6 * time.Hour
		timeFormat = "2006-01-02T15:00:00Z"
	} else if duration <= 30*24*time.Hour {
		interval = 24 * time.Hour
		timeFormat = "2006-01-02T00:00:00Z"
	} else {
		interval = 7 * 24 * time.Hour
		timeFormat = "2006-01-02T00:00:00Z"
	}

	// Group alerts by time intervals
	timeCounts := make(map[string]int)

	for _, alert := range alerts {
		if alert.CreatedAt.After(start) && alert.CreatedAt.Before(end) {
			// Truncate time to the interval
			truncatedTime := alert.CreatedAt.Truncate(interval)
			
			// For longer intervals, adjust to start of day/week
			switch interval {
			case 24 * time.Hour:
				truncatedTime = time.Date(truncatedTime.Year(), truncatedTime.Month(), truncatedTime.Day(), 0, 0, 0, 0, truncatedTime.Location())
			case 7 * 24 * time.Hour:
				// Start of week (Monday)
				weekday := int(truncatedTime.Weekday())
				if weekday == 0 {
					weekday = 7 // Sunday = 7
				}
				truncatedTime = truncatedTime.AddDate(0, 0, -(weekday-1))
				truncatedTime = time.Date(truncatedTime.Year(), truncatedTime.Month(), truncatedTime.Day(), 0, 0, 0, 0, truncatedTime.Location())
			}

			timeKey := truncatedTime.Format(timeFormat)
			timeCounts[timeKey]++
		}
	}

	// Convert to slice and sort by timestamp
	type TimelineResult struct {
		Timestamp time.Time
		Count     int
	}

	var results []TimelineResult
	for timeKey, count := range timeCounts {
		timestamp, err := time.Parse(timeFormat, timeKey)
		if err != nil {
			continue
		}
		results = append(results, TimelineResult{
			Timestamp: timestamp,
			Count:     count,
		})
	}

	// Sort by timestamp
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Timestamp.Before(results[i].Timestamp) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Convert to response format
	timeline := make([]struct {
		Timestamp string `json:"timestamp"`
		Count     int    `json:"count"`
	}, len(results))

	for i, result := range results {
		timeline[i] = struct {
			Timestamp string `json:"timestamp"`
			Count     int    `json:"count"`
		}{
			Timestamp: result.Timestamp.Format(timeFormat),
			Count:     result.Count,
		}
	}

	return timeline, nil
}

// simulateChannelActivity simulates notification activity for a channel
// In a real implementation, you would track actual notifications in a separate table
func (s *statsService) simulateChannelActivity(channel models.NotificationChannel, start, end time.Time) int {
	if !channel.Enabled {
		return 0
	}

	// Simulate activity based on channel type and time range
	hours := int(end.Sub(start).Hours())
	baseActivity := hours / 2 // Base activity of ~0.5 notifications per hour

	// Adjust based on channel type
	switch models.NotificationChannelType(channel.Type) {
	case models.ChannelTypeDingTalk, models.ChannelTypeWeChatWork:
		return baseActivity * 3 // Higher activity for chat channels
	case models.ChannelTypeEmail:
		return baseActivity * 2 // Medium activity for email
	case models.ChannelTypeSMS:
		return baseActivity // Lower activity for SMS (usually for critical alerts only)
	default:
		return baseActivity
	}
}