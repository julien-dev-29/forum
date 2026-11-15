package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julien-dev-29/forum/internal/models"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	posts, err := app.posts.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, http.StatusOK, "home.html", &templateData{
		Posts: posts,
	})

}

func (app *application) postView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	post, err := app.posts.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.render(w, http.StatusOK, "view.html", &templateData{
		Post: post,
	})
}

func (app *application) postViewAll(w http.ResponseWriter, r *http.Request) {
	posts, err := app.posts.GetAll()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, http.StatusOK, "all.html", &templateData{
		Posts: posts,
	})	
}

func (app *application) postCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}
	title := "You can't see me"
	content := "You can't see me \n \n You can't see me"

	id, err := app.posts.Insert(title, content)
	if err != nil {
		app.serverError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", id), http.StatusSeeOther)
}
