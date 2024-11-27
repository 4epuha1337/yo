package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func InitDB() error {
	var err error
	DB, err = sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		return err
	}
	return DB.Ping()
}

func CheckDB() error {
	appDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Ошибка при получении директории приложения: %v", err)
	}

	dbPath := filepath.Join(appDir, "scheduler.db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		DB, err = createDatabase(dbPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func createDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка при создании базы данных: %v", err)
	}
	defer db.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT CHECK(LENGTH(repeat) <= 128)
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func AddTask(date, title, comment, repeat string) (int, error) {
	stmt, err := DB.Prepare("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(date, title, comment, repeat)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func GetTasks() ([]Task, error) {
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	return tasks, nil
}

func GetTaskByID(db *sql.DB, id int64) (*Task, error) {
	query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`
	row := db.QueryRow(query, id)

	var task Task
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("задача с ID %d не найдена", id)
		}
		return nil, err
	}

	return &task, nil
}
