package main

import (
	"net/http"
	"log"

	"github.com/go-chi/chi/v5"
)

var addr = "127.0.0.1:7540"
var webDir = "./web"

func main() {
	r := chi.NewRouter()
	r.Handle("/", http.FileServer(http.Dir(webDir)))
	err := http.ListenAndServe(addr, http.FileServer(http.Dir(webDir)))
	if err != nil {
		log.Panicf("Start server error: %s", err.Error())
	}

	checkDB()
}