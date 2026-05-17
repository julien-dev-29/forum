package handlers

import (
	"html/template"
	"net/http"
)

var postTmpl = template.Must(template.ParseFiles("views/"))

func HandlePost(w http.ResponseWriter, r *http.Request) {

}
