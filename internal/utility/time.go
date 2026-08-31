package utility

import "time"

func GetTodayStart() string {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return start.Format(time.RFC3339)
}

func GetTodayEnd() string {
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	return end.Format(time.RFC3339)
}

func GetPastDaysStart(days int) string {
	now := time.Now().UTC()
	past := now.AddDate(0, 0, -days)
	start := time.Date(past.Year(), past.Month(), past.Day(), 0, 0, 0, 0, time.UTC)
	return start.Format(time.RFC3339)
}
