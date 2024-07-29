package provider

import (
	"errors"
	"strconv"
	"time"
)

func parseDuration(durationStr string) (time.Duration, error) {
	if len(durationStr) < 2 {
		return 0, errors.New("invalid duration string")
	}

	unit := durationStr[len(durationStr)-1]
	valueStr := durationStr[:len(durationStr)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, errors.New("invalid number in duration string")
	}

	var duration time.Duration
	switch unit {
	case 's':
		duration = time.Duration(value) * time.Second
	case 'm':
		duration = time.Duration(value) * time.Minute
	case 'h':
		duration = time.Duration(value) * time.Hour
	case 'd':
		duration = time.Duration(value) * time.Hour * 24
	default:
		return 0, errors.New("invalid unit in duration string")
	}

	return duration, nil
}

func formatDuration(duration time.Duration) string {
	seconds := int64(duration.Seconds())
	var value int64
	var unit string

	switch {
	case seconds == 0:
		value = 0
		unit = "s"
	case seconds%86400 == 0:
		value = seconds / 86400
		unit = "d"
	case seconds%3600 == 0:
		value = seconds / 3600
		unit = "h"
	case seconds%60 == 0:
		value = seconds / 60
		unit = "m"
	default:
		value = seconds
		unit = "s"
	}

	return strconv.FormatInt(value, 10) + unit
}
