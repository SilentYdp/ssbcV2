package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		name = "Guest"
	}
	log.Printf("Received request for %s.\n", name)
	w.Write([]byte(fmt.Sprintf("Hello, %s!\n", name)))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	server := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("Starting Server.")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
