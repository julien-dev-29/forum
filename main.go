package main

import (
	"fmt"
	"main/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	mux := mux.NewRouter()

	mux.HandleFunc("/", handlers.HandleHome)

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("http://localhost:8000")
	http.ListenAndServe(":8000", mux)
}
