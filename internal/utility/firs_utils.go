package utility

import (
	"fmt"
	"time"
)

func GetString(data map[string]interface{}, key string) (string, bool) {
	if val, ok := data[key]; ok {
		if s, ok := val.(string); ok {
			return s, true
		}
	}
	return "", false
}

func GetBool(data map[string]interface{}, key string) (bool, bool) {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func GetInt(data map[string]interface{}, key string) (int, bool) {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case int:
			return v, true
		case float64:
			return int(v), true
		}
	}
	return 0, false
}

func GetFloat(data map[string]interface{}, key string) (float64, bool) {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f, true
		}
	}
	return 0, false
}

func GetTime(data map[string]interface{}, key string) (time.Time, error) {
	s, ok := GetString(data, key)
	if !ok || s == "" {
		return time.Time{}, fmt.Errorf("missing or invalid time string for key: %s", key)
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time from value: %s", s)
}
