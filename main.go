package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/4epuha1337/yo/api"
	"github.com/4epuha1337/yo/db"
)

var webDir = "./web"

func main() {
	err := db.CheckDB()
	if err != nil {
		log.Panicf("Database error: %s", err.Error())
	}
	err = db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.DB.Close()
	r := chi.NewRouter()

	r.Get("/api/nextdate", api.NextDateHandler)
	r.Get("/api/tasks", api.GetTasks)
	r.Post("/api/task", api.PostTask)
	r.Get("/api/task", api.GetTask)
	r.Put("/api/task", api.UpdateTaskHandler)
	r.Post("/api/task/done", api.MarkTaskDone)
	r.Delete("/api/task", api.DeleteTask)

	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir(webDir))))
	log.Println("Server is running on port 7540...")
	err = http.ListenAndServe(":7540", r)
	if err != nil {
		log.Printf("Error starting server: %v\n", err)
	}
}
