package main

import (
	"fmt"
	"log"
	"main/database"
	"main/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	err = database.CreateTables()
	if err != nil {
		log.Fatal(err)
	}

	err = database.SeedPosts()
	if err != nil {
		log.Fatal(err)
	}
	mux := mux.NewRouter()

	mux.HandleFunc("/", handlers.Home)

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	fmt.Println("http://localhost:8000")
	http.ListenAndServe(":8000", mux)
}
