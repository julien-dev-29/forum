package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	if err := os.MkdirAll(filepath.Join("ui", "static", "uploads"), 0755); err != nil {
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
	mux.HandleFunc("GET /auth/google/login", h.LoginGoogle)
	mux.HandleFunc("GET /auth/google/callback", h.CallbackGoogle)
	mux.HandleFunc("GET /auth/github/login", h.LoginGitHub)
	mux.HandleFunc("GET /auth/github/callback", h.CallbackGitHub)
	mux.HandleFunc("GET /post/new", h.CreatePostGet)
	mux.HandleFunc("POST /post/new", h.CreatePostPost)
	mux.HandleFunc("GET /post", h.ViewPost)
	mux.HandleFunc("POST /comment", h.CreateComment)
	mux.HandleFunc("POST /like/post", h.LikePost)
	mux.HandleFunc("POST /like/comment", h.LikeComment)
	mux.HandleFunc("GET /activity", h.ActivityShow)
	mux.HandleFunc("GET /notifications", h.NotifList)
	mux.HandleFunc("POST /notifications/read", h.NotifRead)
	mux.HandleFunc("GET /api/notifications/stream", h.NotifStream)
	mux.HandleFunc("GET /post/edit", h.EditPostGet)
	mux.HandleFunc("POST /post/edit", h.EditPostPost)
	mux.HandleFunc("POST /post/delete", h.DeletePost)
	mux.HandleFunc("GET /comment/edit", h.EditCommentGet)
	mux.HandleFunc("POST /comment/edit", h.EditCommentPost)
	mux.HandleFunc("POST /comment/delete", h.DeleteComment)

	// Admin
	mux.HandleFunc("GET /admin", h.AdminDashboard)
	mux.HandleFunc("POST /admin/users/promote", h.AdminPromoteUser)
	mux.HandleFunc("POST /admin/users/demote", h.AdminDemoteUser)
	mux.HandleFunc("POST /admin/reports/respond", h.AdminRespondReport)
	mux.HandleFunc("POST /admin/categories/create", h.AdminCreateCategory)
	mux.HandleFunc("POST /admin/categories/delete", h.AdminDeleteCategory)
	mux.HandleFunc("POST /admin/mod-requests/approve", h.AdminApproveModRequest)
	mux.HandleFunc("POST /admin/mod-requests/deny", h.AdminDenyModRequest)

	// Moderator
	mux.HandleFunc("POST /mod/report/post", h.ModReportPost)
	mux.HandleFunc("POST /mod/report/comment", h.ModReportComment)
	mux.HandleFunc("POST /mod/delete/post", h.ModDeletePost)
	mux.HandleFunc("POST /mod/delete/comment", h.ModDeleteComment)

	// Mod request
	mux.HandleFunc("GET /mod/request", h.ModRequestGet)
	mux.HandleFunc("POST /mod/request", h.ModRequestPost)

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
