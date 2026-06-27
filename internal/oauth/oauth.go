package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserURL      string
	UserEmailURL string
	Scopes       []string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GoogleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func GoogleConfig(clientID, clientSecret, redirectURL string) *Config {
	return &Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserURL:      "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:       []string{"email", "profile"},
	}
}

func GitHubConfig(clientID, clientSecret, redirectURL string) *Config {
	return &Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserURL:      "https://api.github.com/user",
		UserEmailURL: "https://api.github.com/user/emails",
		Scopes:       []string{"read:user", "user:email"},
	}
}

func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func SetStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/auth/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
}

func VerifyStateCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	c, err := r.Cookie("oauth_state")
	if err != nil {
		return "", fmt.Errorf("state cookie not found")
	}
	state := c.Value
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/auth/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return state, nil
}

func GetAuthURL(cfg *Config, state string) string {
	return fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
		cfg.AuthURL,
		url.QueryEscape(cfg.ClientID),
		url.QueryEscape(cfg.RedirectURL),
		url.QueryEscape(strings.Join(cfg.Scopes, " ")),
		url.QueryEscape(state),
	)
}

func ExchangeCode(cfg *Config, code string) (string, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequest("POST", cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tr.AccessToken == "" {
		return "", fmt.Errorf("no access token in response: %s", string(body))
	}

	return tr.AccessToken, nil
}

func fetchJSON(url, token string, v any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func GetGoogleUser(token string) (id, email, name string, err error) {
	var gu GoogleUser
	if err := fetchJSON("https://www.googleapis.com/oauth2/v2/userinfo", token, &gu); err != nil {
		return "", "", "", fmt.Errorf("get google user: %w", err)
	}
	return gu.ID, gu.Email, gu.Name, nil
}

func GetGitHubUser(token string) (id, email, name string, err error) {
	var gu GitHubUser
	if err := fetchJSON("https://api.github.com/user", token, &gu); err != nil {
		return "", "", "", fmt.Errorf("get github user: %w", err)
	}

	// GitHub may not return email in the user endpoint if it's private
	if gu.Email == "" {
		var emails []GitHubEmail
		if err := fetchJSON("https://api.github.com/user/emails", token, &emails); err != nil {
			return "", "", "", fmt.Errorf("get github emails: %w", err)
		}
		for _, e := range emails {
			if e.Primary && e.Verified {
				gu.Email = e.Email
				break
			}
		}
	}

	githubID := fmt.Sprintf("%d", gu.ID)
	displayName := gu.Name
	if displayName == "" {
		displayName = gu.Login
	}

	return githubID, gu.Email, displayName, nil
}
