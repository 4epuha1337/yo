package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var dateFormat = "20060102"

func NextDate(now time.Time, date string, repeat string) (string, error) {
	taskDate, err := time.Parse(dateFormat, date)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %v", err)
	}

	if repeat == "" {
		return "", errors.New("repetition rule is missing")
	}

	parts := strings.Split(repeat, " ")
	var nextDate time.Time
	switch parts[0] {
	case "d":
		if len(parts) != 2 {
			return "", errors.New("invalid repetition rule for 'd', missing number of days")
		}
		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 || days > 400 {
			return "", errors.New("invalid number of days in 'd' rule, should be between 1 and 400")
		}
		nextDate = taskDate.AddDate(0, 0, days)

	case "y":
		nextDate = taskDate.AddDate(1, 0, 0)

	default:
		return "", errors.New("unsupported repetition rule: " + parts[0])
	}

	for nextDate.Before(now) {
		if parts[0] == "d" {
			days, _ := strconv.Atoi(parts[1])
			nextDate = nextDate.AddDate(0, 0, days)
		} else if parts[0] == "y" {
			nextDate = nextDate.AddDate(1, 0, 0)
		}
	}

	return nextDate.Format(dateFormat), nil
}
