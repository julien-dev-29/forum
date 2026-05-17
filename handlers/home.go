package handlers

import (
	"html/template"
	"net/http"
)

var homeTmpl = template.Must(template.ParseFiles("views/index.html"))

func HandleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"Title": "Yolo les kikis",
	}
	homeTmpl.Execute(w, data)
}
