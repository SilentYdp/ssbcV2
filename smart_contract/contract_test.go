package smart_contract

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBuildAndRun(t *testing.T) {
	BuildAndRun("test/","test")
}
func TestA(t *testing.T) {
	a:= struct {
		Code string
	}{
		Code:`package main

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
`}
	aB,_:=json.Marshal(a)
	fmt.Println(string(aB))
}

func TestListAllContains(t *testing.T) {
	cs:=ListAllContains()
	for _, container := range cs {
		fmt.Println(container.ID, container.Image)
		for _,p:=range container.Ports{
			fmt.Println(p.PublicPort)
		}
	}

}
