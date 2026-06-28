package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forum/internal/csrf"
	"forum/internal/database/sqlite"
	"forum/internal/handlers"
	"forum/internal/middleware"
	"forum/internal/oauth"
	"forum/internal/ratelimit"
	"forum/internal/session"
	"forum/internal/tls"

	"golang.org/x/crypto/acme/autocert"
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

	if err := initEncryption(); err != nil {
		log.Fatalf("encryption: %v", err)
	}

	handlers.InitTemplates()

	h := handlers.New(db)

	rateLimitLogin := ratelimit.New(5, time.Minute, 5)
	rateLimitRegister := ratelimit.New(3, time.Minute, 3)
	rateLimitPost := ratelimit.New(30, time.Minute, 30)
	rateLimitAPI := ratelimit.New(60, time.Minute, 60)
	rateLimitAuthAPI := ratelimit.New(10, time.Minute, 10)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /register", h.RegisterGet)
	mux.Handle("POST /register", rateLimitRegister.Middleware()(http.HandlerFunc(h.RegisterPost)))
	mux.HandleFunc("GET /login", h.LoginGet)
	mux.Handle("POST /login", rateLimitLogin.Middleware()(http.HandlerFunc(h.LoginPost)))
	mux.HandleFunc("POST /logout", h.LogoutPost)
	mux.Handle("GET /auth/google/login", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.LoginGoogle)))
	mux.Handle("GET /auth/google/callback", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.CallbackGoogle)))
	mux.Handle("GET /auth/github/login", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.LoginGitHub)))
	mux.Handle("GET /auth/github/callback", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.CallbackGitHub)))
	mux.HandleFunc("GET /post/new", h.CreatePostGet)
	mux.Handle("POST /post/new", rateLimitPost.Middleware()(http.HandlerFunc(h.CreatePostPost)))
	mux.HandleFunc("GET /post", h.ViewPost)
	mux.Handle("POST /comment", rateLimitPost.Middleware()(http.HandlerFunc(h.CreateComment)))
	mux.Handle("POST /like/post", rateLimitAPI.Middleware()(http.HandlerFunc(h.LikePost)))
	mux.Handle("POST /like/comment", rateLimitAPI.Middleware()(http.HandlerFunc(h.LikeComment)))
	mux.HandleFunc("GET /activity", h.ActivityShow)
	mux.HandleFunc("GET /notifications", h.NotifList)
	mux.Handle("POST /notifications/read", rateLimitAPI.Middleware()(http.HandlerFunc(h.NotifRead)))
	mux.Handle("GET /api/notifications/stream", rateLimitAPI.Middleware()(http.HandlerFunc(h.NotifStream)))
	mux.HandleFunc("GET /post/edit", h.EditPostGet)
	mux.Handle("POST /post/edit", rateLimitPost.Middleware()(http.HandlerFunc(h.EditPostPost)))
	mux.Handle("POST /post/delete", rateLimitPost.Middleware()(http.HandlerFunc(h.DeletePost)))
	mux.HandleFunc("GET /comment/edit", h.EditCommentGet)
	mux.Handle("POST /comment/edit", rateLimitPost.Middleware()(http.HandlerFunc(h.EditCommentPost)))
	mux.Handle("POST /comment/delete", rateLimitPost.Middleware()(http.HandlerFunc(h.DeleteComment)))
	mux.HandleFunc("GET /admin", h.AdminDashboard)
	mux.Handle("POST /admin/users/promote", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminPromoteUser)))
	mux.Handle("POST /admin/users/demote", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminDemoteUser)))
	mux.Handle("POST /admin/reports/respond", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminRespondReport)))
	mux.Handle("POST /admin/categories/create", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminCreateCategory)))
	mux.Handle("POST /admin/categories/delete", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminDeleteCategory)))
	mux.Handle("POST /admin/mod-requests/approve", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminApproveModRequest)))
	mux.Handle("POST /admin/mod-requests/deny", rateLimitAuthAPI.Middleware()(http.HandlerFunc(h.AdminDenyModRequest)))
	mux.Handle("POST /mod/report/post", rateLimitAPI.Middleware()(http.HandlerFunc(h.ModReportPost)))
	mux.Handle("POST /mod/report/comment", rateLimitAPI.Middleware()(http.HandlerFunc(h.ModReportComment)))
	mux.Handle("POST /mod/delete/post", rateLimitPost.Middleware()(http.HandlerFunc(h.ModDeletePost)))
	mux.Handle("POST /mod/delete/comment", rateLimitPost.Middleware()(http.HandlerFunc(h.ModDeleteComment)))
	mux.HandleFunc("GET /mod/request", h.ModRequestGet)
	mux.Handle("POST /mod/request", rateLimitPost.Middleware()(http.HandlerFunc(h.ModRequestPost)))

	fs := http.FileServer(http.Dir("ui/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	authMW := middleware.Auth(db, mux)
	csrfMW := csrf.Middleware(authMW)
	handler := tls.SecurityHeaders(csrfMW)

	domain := os.Getenv("AUTOCERT_HOST")
	devMode := os.Getenv("DEV_MODE") == "true"
	port := os.Getenv("PORT")

	if domain != "" {
		if port == "" {
			port = "443"
		}
		setSecure(true)
		handler = tls.HSTS(handler)

		cacheDir := os.Getenv("AUTOCERT_CACHE_DIR")
		if cacheDir == "" {
			cacheDir = "certs"
		}

		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(strings.Split(domain, ",")...),
			Cache:      autocert.DirCache(cacheDir),
		}

		go func() {
			httpHandler := m.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target := "https://" + r.Host + r.URL.String()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
			}))
			log.Printf("HTTP ACME/redirect server on :80")
			if err := http.ListenAndServe(":80", httpHandler); err != nil {
				log.Printf("HTTP server: %v", err)
			}
		}()

		srv := &http.Server{
			Addr:      ":" + port,
			Handler:   handler,
			TLSConfig: tls.NewTLSConfig(m, false),
		}
		log.Printf("HTTPS server starting on :%s", port)
		log.Fatal(srv.ListenAndServeTLS("", ""))
	} else if devMode {
		if port == "" {
			port = "8443"
		}
		setSecure(true)

		srv := &http.Server{
			Addr:      ":" + port,
			Handler:   handler,
			TLSConfig: tls.NewTLSConfig(nil, true),
		}
		log.Printf("HTTPS (dev, self-signed) server starting on :%s", port)
		log.Fatal(srv.ListenAndServeTLS("", ""))
	} else {
		if port == "" {
			port = "8081"
		}
		setSecure(false)
		log.Printf("HTTP server starting on :%s", port)
		log.Fatal(http.ListenAndServe(":"+port, handler))
	}
}

func setSecure(v bool) {
	session.Configure(v)
	oauth.Configure(v)
	csrf.Configure(v)
}

func initEncryption() error {
	key := os.Getenv("DB_ENCRYPTION_KEY")
	if key == "" {
		return nil
	}
	return sqlite.InitEncryption(key)
}
