package handlers

import (
	"html/template"
	"net/http"
)

var postTmpl = template.Must(template.ParseFiles("views/base.html"))

func HandlePost(w http.ResponseWriter, r *http.Request) {
	postTmpl.Execute(w, nil)
}
