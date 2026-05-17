package handlers

import (
	"html/template"
	"net/http"
)

var homeTmpl = template.Must(template.ParseFiles("views/index.html"))

func HandleHome(w http.ResponseWriter, r *http.Request) {
	homeTmpl.Execute(w, nil)
}
