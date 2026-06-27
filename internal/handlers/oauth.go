package handlers

import (
	"database/sql"
	"net/http"
	"os"

	"forum/internal/database/sqlite"
	"forum/internal/oauth"
	"forum/internal/session"
)

type oauthHandler struct {
	db *sql.DB
}

func (h *oauthHandler) loginGoogle(w http.ResponseWriter, r *http.Request) {
	if isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}
	oauth.SetStateCookie(w, state)

	cfg := oauth.GoogleConfig(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		redirectURL(r, "/auth/google/callback"),
	)
	http.Redirect(w, r, oauth.GetAuthURL(cfg, state), http.StatusFound)
}

func (h *oauthHandler) callbackGoogle(w http.ResponseWriter, r *http.Request) {
	state, err := oauth.VerifyStateCookie(w, r)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state {
		renderError(w, http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		renderError(w, http.StatusBadRequest)
		return
	}

	cfg := oauth.GoogleConfig(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		redirectURL(r, "/auth/google/callback"),
	)

	token, err := oauth.ExchangeCode(cfg, code)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	oauthID, email, name, err := oauth.GetGoogleUser(token)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	h.oauthLogin(w, r, "google", oauthID, email, name)
}

func (h *oauthHandler) loginGitHub(w http.ResponseWriter, r *http.Request) {
	if isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}
	oauth.SetStateCookie(w, state)

	cfg := oauth.GitHubConfig(
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		redirectURL(r, "/auth/github/callback"),
	)
	http.Redirect(w, r, oauth.GetAuthURL(cfg, state), http.StatusFound)
}

func (h *oauthHandler) callbackGitHub(w http.ResponseWriter, r *http.Request) {
	state, err := oauth.VerifyStateCookie(w, r)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state {
		renderError(w, http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		renderError(w, http.StatusBadRequest)
		return
	}

	cfg := oauth.GitHubConfig(
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		redirectURL(r, "/auth/github/callback"),
	)

	token, err := oauth.ExchangeCode(cfg, code)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	oauthID, email, name, err := oauth.GetGitHubUser(token)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	h.oauthLogin(w, r, "github", oauthID, email, name)
}

func (h *oauthHandler) oauthLogin(w http.ResponseWriter, r *http.Request, provider, oauthID, email, name string) {
	if email == "" {
		renderTemplate(w, "login.html", map[string]any{
			"Authenticated": false,
			"Error":         "Could not retrieve email from " + provider + ". Please make sure your email is public.",
		})
		return
	}

	user, err := sqlite.GetUserByOAuth(h.db, provider, oauthID)
	if err == nil {
		h.createSessionAndRedirect(w, r, user.ID)
		return
	}

	user, err = sqlite.GetUserByEmail(h.db, email)
	if err == nil {
		if err := sqlite.SetUserOAuth(h.db, user.ID, provider, oauthID); err != nil {
			renderError(w, http.StatusInternalServerError)
			return
		}
		h.createSessionAndRedirect(w, r, user.ID)
		return
	}

	username := name
	if username == "" {
		username = provider + "_" + oauthID
		if len(username) > 20 {
			username = username[:20]
		}
	}

	user, err = sqlite.CreateOAuthUser(h.db, email, username, provider, oauthID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	h.createSessionAndRedirect(w, r, user.ID)
}

func (h *oauthHandler) createSessionAndRedirect(w http.ResponseWriter, r *http.Request, userID int64) {
	token, err := session.Create(h.db, userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}
	session.WriteCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func redirectURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	return scheme + "://" + r.Host + path
}
