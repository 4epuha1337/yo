package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/4epuha1337/yo/db"
)

var dateFormat = "20060102"

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowParam := r.FormValue("now")
	dateParam := r.FormValue("date")
	repeatParam := r.FormValue("repeat")

	if nowParam == "" || dateParam == "" || repeatParam == "" {
		writeErrorResponse(w, "Missing required query parameters: now, date, repeat", http.StatusBadRequest)
		return
	}

	now, err := time.Parse(dateFormat, nowParam)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("Invalid now parameter: %v", err), http.StatusBadRequest)
		return
	}

	nextDate, err := db.NextDate(now, dateParam, repeatParam)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to calculate next date: %s", err.Error()), http.StatusBadRequest)
		return
	}

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
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := validateTitle(req.Title); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	date, err := validateDate(req.Date)
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Date = date

	today := time.Now()
	taskDate, _ := time.Parse(dateFormat, req.Date)
	if beforeDatesWithoutTime(taskDate, today) {
		if req.Repeat == "" {
			req.Date = today.Format(dateFormat)
		} else {
			nextDate, err := db.NextDate(today, req.Date, req.Repeat)
			if err != nil {
				writeErrorResponse(w, fmt.Sprintf("invalid repeat rule: %s", err.Error()), http.StatusBadRequest)
				return
			}
			req.Date = nextDate
		}
	}

	id, err := db.AddTask(req.Date, req.Title, req.Comment, req.Repeat)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("failed to add task: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, ApiResponse{ID: strconv.Itoa(id)})
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := db.GetTasks()
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("Error fetching tasks: %s", err), http.StatusInternalServerError)
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
	writeJSONResponse(w, response)
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

	task, err := db.GetTaskByID(string(id))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req db.Task
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := validateTaskID(req.ID); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateTitle(req.Title); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	date, err := validateDate(req.Date)
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Date = date

	if err := validateRepeatRule(req.Repeat); err != nil {
		writeErrorResponse(w, fmt.Sprintf("invalid repeat rule: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if !db.TaskExists(req.ID) {
		writeErrorResponse(w, "task not found", http.StatusNotFound)
		return
	}

	if err := db.UpdateTask(req); err != nil {
		writeErrorResponse(w, "failed to update task", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, struct{}{})
}

func MarkTaskDone(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseIDParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := db.GetTaskByID(strconv.Itoa(int(taskID)))
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorResponse(w, "task not found", http.StatusNotFound)
		} else {
			writeErrorResponse(w, fmt.Sprintf("failed to fetch task: %s", err.Error()), http.StatusInternalServerError)
		}
		return
	}

	if task.Repeat == "" {
		if _, err := db.DeleteTaskByID(strconv.Itoa(int(taskID))); err != nil {
			writeErrorResponse(w, fmt.Sprintf("failed to delete task: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		writeJSONResponse(w, struct{}{})
		return
	}

	today := time.Now()
	nextDate, err := db.NextDate(today, task.Date, task.Repeat)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("failed to calculate next date: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if err := db.UpdateTaskDate(task.ID, nextDate); err != nil {
		writeErrorResponse(w, fmt.Sprintf("failed to update task: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, struct{}{})
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseIDParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	rowsAffected, err := db.DeleteTaskByID(strconv.Itoa(int(taskID)))
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("failed to delete task: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeErrorResponse(w, "task not found", http.StatusNotFound)
		return
	}

	writeJSONResponse(w, struct{}{})
}