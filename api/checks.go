package api

import (
	"time"
	"fmt"
	"strings"
	"encoding/json"
	"net/http"
	"strconv"
)

func validateDate(date string) (string, error) {
	if date == "" {
		return time.Now().Format(dateFormat), nil
	}
	if _, err := time.Parse(dateFormat, date); err != nil {
		return "", fmt.Errorf("invalid date format, expected YYYYMMDD")
	}
	return date, nil
}

func validateTaskID(taskID string) error {
	if strings.TrimSpace(taskID) == "" || taskID == "0" {
		return fmt.Errorf("task ID is required")
	}
	return nil
}

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

func writeErrorResponse(w http.ResponseWriter, message string, status int) {
	http.Error(w, fmt.Sprintf(`{"error":"%s"}`, message), status)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func parseIDParam(r *http.Request, param string) (int64, error) {
	idStr := r.URL.Query().Get(param)
	if idStr == "" {
		return 0, fmt.Errorf("missing required parameter: %s", param)
	}
	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format for parameter: %s", param)
	}
	return id, nil
}

func validateRepeatRule(repeat string) error {
	if repeat == "" {
		return nil
	}

	parts := strings.Split(repeat, " ")
	if len(parts) > 2 || len(parts) < 1 {
		return fmt.Errorf("invalid repeat rule format")
	}

	switch parts[0] {
	case "d":
		if len(parts) != 2 {
			return fmt.Errorf("missing interval for daily repeat")
		}
		interval, err := strconv.Atoi(parts[1])
		if err != nil || interval <= 0 {
			return fmt.Errorf("invalid daily repeat interval")
		}
	case "y":
		if len(parts) != 1 {
			return fmt.Errorf("yearly repeat rule should not have additional parameters")
		}
	default:
		return fmt.Errorf("unsupported repeat type: %s", parts[0])
	}

	return nil
}