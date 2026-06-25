package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"forum/internal/database/sqlite"
	"forum/internal/handlers"
	"forum/internal/middleware"
	"forum/internal/session"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "forum.db"
	}

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := sqlite.InitSchema(db); err != nil {
		log.Fatal(err)
	}

	if err := sqlite.SeedCategories(db); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			session.CleanupExpired(db)
			time.Sleep(30 * time.Minute)
		}
	}()

	handlers.InitTemplates()

	h := handlers.New(db)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /register", h.RegisterGet)
	mux.HandleFunc("POST /register", h.RegisterPost)
	mux.HandleFunc("GET /login", h.LoginGet)
	mux.HandleFunc("POST /login", h.LoginPost)
	mux.HandleFunc("POST /logout", h.LogoutPost)
	mux.HandleFunc("GET /post/new", h.CreatePostGet)
	mux.HandleFunc("POST /post/new", h.CreatePostPost)
	mux.HandleFunc("GET /post", h.ViewPost)
	mux.HandleFunc("POST /comment", h.CreateComment)
	mux.HandleFunc("POST /like/post", h.LikePost)
	mux.HandleFunc("POST /like/comment", h.LikeComment)

	fs := http.FileServer(http.Dir("ui/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	handler := middleware.Auth(db, mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
