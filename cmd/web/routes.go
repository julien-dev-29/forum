package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))

	mux.Handle("/static/", http.StripPrefix("/static", neuter(fileServer)))

	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/post/view", app.postView)
	mux.HandleFunc("/post/all", app.postViewAll)
	mux.HandleFunc("/post/create", app.postCreate)

	return mux
}
