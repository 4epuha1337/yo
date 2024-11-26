package api

import (
	"net/http"
	"strconv"
	"strings"

	"encoding/json"
	"fmt"
	"time"

	"github.com/4epuha1337/yo/db"
)

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
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// Функция для обработки POST-запроса
func PostTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	// Читаем и парсим JSON-запрос
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error":"invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	// Проверяем наличие обязательного поля title
	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	// Если date пустая, устанавливаем сегодняшнюю дату
	if req.Date == "" {
		req.Date = time.Now().Format("20060102")
	} else {
		// Проверяем формат даты
		_, err := time.Parse("20060102", req.Date)
		if err != nil {
			http.Error(w, `{"error":"invalid date format, expected YYYYMMDD"}`, http.StatusBadRequest)
			return
		}
	}

	// Обработка даты задачи
	today := time.Now()
	taskDate, _ := time.Parse("20060102", req.Date)

	if taskDate.Before(today) {
		if req.Repeat == "" {
			// Если правило повторения отсутствует, ставим сегодняшнюю дату
			req.Date = today.Format("20060102")
		} else {
			// Если есть правило повторения, используем NextDate
			nextDate, err := db.NextDate(today, req.Date, req.Repeat)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"invalid repeat rule: %s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			req.Date = nextDate
		}
	}

	// Добавляем задачу в базу данных
	id, err := db.AddTask(req.Date, req.Title, req.Comment, req.Repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to add task: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ с ID задачи
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ApiResponse{ID: strconv.Itoa(id)})
}
