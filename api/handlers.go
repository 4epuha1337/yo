package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"encoding/json"
	"fmt"
	"time"

	"github.com/4epuha1337/yo/db"
)

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

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowParam := r.FormValue("now")
	dateParam := r.FormValue("date")
	repeatParam := r.FormValue("repeat")

	if nowParam == "" || dateParam == "" || repeatParam == "" {
		http.Error(w, "Missing required query parameters: now, date, repeat", http.StatusBadRequest)
		return
	}

	now, err := time.Parse("20060102", nowParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid now parameter: %v", err), http.StatusBadRequest)
		return
	}

	nextDate, err := db.NextDate(now, dateParam, repeatParam)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

type TaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func beforeDatesWithoutTime(date1, date2 time.Time) bool {
	date1 = date1.Truncate(24 * time.Hour)
	date2 = date2.Truncate(24 * time.Hour)

	return date1.Before(date2)
}

type ApiResponse struct {
	Tasks []db.Task `json:"tasks"`
	ID    string    `json:"id,omitempty"`
	Error string    `json:"error,omitempty"`
}

func PostTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error":"invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	if req.Date == "" {
		req.Date = time.Now().Format("20060102")
	} else {
		_, err := time.Parse("20060102", req.Date)
		if err != nil {
			http.Error(w, `{"error":"invalid date format, expected YYYYMMDD"}`, http.StatusBadRequest)
			return
		}
	}

	today := time.Now()
	taskDate, _ := time.Parse("20060102", req.Date)

	if beforeDatesWithoutTime(taskDate, today) {
		if req.Repeat == "" {
			req.Date = today.Format("20060102")
		} else {
			nextDate, err := db.NextDate(today, req.Date, req.Repeat)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"invalid repeat rule: %s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			req.Date = nextDate
		}
	}

	id, err := db.AddTask(req.Date, req.Title, req.Comment, req.Repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to add task: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ApiResponse{ID: strconv.Itoa(id)})
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := db.GetTasks() 
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching tasks: %s", err), http.StatusInternalServerError)
		return
	}

	var response ApiResponse
	for _, task := range tasks {
		response.Tasks = append(response.Tasks, db.Task{
			ID:      task.ID,
			Date:    task.Date,
			Title:   task.Title,
			Comment: task.Comment,
			Repeat:  task.Repeat,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if len(response.Tasks) == 0 {
		response.Tasks = []db.Task{} 
	}
	json.NewEncoder(w).Encode(response)
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	ids := r.URL.Query().Get("id")
	if ids == "" {
		http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	var id int64
	_, err := fmt.Sscanf(ids, "%d", &id)
	if err != nil {
		http.Error(w, `{"error":"Неверный формат идентификатора"}`, http.StatusBadRequest)
		return
	}

	task, err := db.GetTaskByID(db.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req db.Task
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error":"invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	if req.ID == "0" {
		http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	if req.Date == "" {
		req.Date = time.Now().Format("20060102")
	} else {
		_, err := time.Parse("20060102", req.Date)
		if err != nil {
			http.Error(w, `{"error":"invalid date format, expected YYYYMMDD"}`, http.StatusBadRequest)
			return
		}
	}

	err = validateRepeatRule(req.Repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid repeat rule: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	var exists bool
	err = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM scheduler WHERE id = ?)`, req.ID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE scheduler
		SET date = ?, title = ?, comment = ?, repeat = ?
		WHERE id = ?`,
		req.Date, req.Title, req.Comment, req.Repeat, req.ID)
	if err != nil {
		http.Error(w, `{"error":"failed to update task"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func MarkTaskDone(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Task ID is required"}`, http.StatusBadRequest)
		return
	}

	var task struct {
		ID      int64  `db:"id"`
		Date    string `db:"date"`
		Repeat  string `db:"repeat"`
		Title   string `db:"title"`
		Comment string `db:"comment"`
	}
	err := db.DB.QueryRow("SELECT id, date, repeat, title, comment FROM scheduler WHERE id = ?", taskID).Scan(&task.ID, &task.Date, &task.Repeat, &task.Title, &task.Comment)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error":"Failed to fetch task: %s"}`, err.Error()), http.StatusInternalServerError)
		}
		return
	}

	if task.Repeat == "" {
		_, err := db.DB.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Failed to delete task: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}")) 
		return
	}

	today := time.Now()
	nextDate, err := db.NextDate(today, task.Date, task.Repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to calculate next date: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, task.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to update task: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Task ID is required"}`, http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to delete task: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to verify deletion: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}
