package handlers

import (
	"html/template"
	"main/database"
	"net/http"
)

type HomeData struct {
	Title string
	Posts []database.Post
}

var homeTmpl = template.Must(template.ParseFiles(
	"views/base.html",
	"views/header.html",
	"views/footer.html",
	"views/index.html",
	"views/sidebar.html",
))

func Home(w http.ResponseWriter, r *http.Request) {
	posts, err := database.GetPosts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := HomeData{
		Title: "Forum",
		Posts: posts,
	}

	homeTmpl.ExecuteTemplate(w, "base.html", data)
}
