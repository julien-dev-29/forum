package csrf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	t1, err := GenerateToken()
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}
	if len(t1) == 0 {
		t.Fatal("expected non-empty token")
	}

	t2, err := GenerateToken()
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}
	if t1 == t2 {
		t.Fatal("expected different tokens on each call")
	}
}

func TestCookieRoundTrip(t *testing.T) {
	Secure = false

	token, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	SetCookie(w, token)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])

	got := GetToken(req)
	if got != token {
		t.Fatalf("expected token %q, got %q", token, got)
	}
}

func TestValidate_FormValue(t *testing.T) {
	Secure = false

	token, _ := GenerateToken()

	body := strings.NewReader("csrf_token=" + token)
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	SetCookie(w, token)
	req.AddCookie(w.Result().Cookies()[0])

	if !Validate(req) {
		t.Fatal("expected valid CSRF token from form")
	}
}

func TestValidate_Header(t *testing.T) {
	Secure = false

	token, _ := GenerateToken()
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set(headerName, token)

	w := httptest.NewRecorder()
	SetCookie(w, token)
	req.AddCookie(w.Result().Cookies()[0])

	if !Validate(req) {
		t.Fatal("expected valid CSRF token from header")
	}
}

func TestValidate_Invalid(t *testing.T) {
	Secure = false

	token, _ := GenerateToken()
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set(headerName, "wrong-token")

	w := httptest.NewRecorder()
	SetCookie(w, token)
	req.AddCookie(w.Result().Cookies()[0])

	if Validate(req) {
		t.Fatal("expected invalid CSRF token")
	}
}

func TestValidate_MissingCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set(headerName, "some-token")

	if Validate(req) {
		t.Fatal("expected invalid with missing cookie")
	}
}

func TestValidate_EmptyToken(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "cookie-token"})

	if Validate(req) {
		t.Fatal("expected invalid with empty request token")
	}
}

func TestMiddleware_SkipGET(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d", rr.Code)
	}
}

func TestMiddleware_BlockPOST(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST without token, got %d", rr.Code)
	}
}

func TestMiddleware_AllowPOSTWithToken(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token, _ := GenerateToken()

	w := httptest.NewRecorder()
	SetCookie(w, token)

	req := httptest.NewRequest("POST", "/", strings.NewReader("csrf_token="+token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(w.Result().Cookies()[0])

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST with valid token, got %d", rr.Code)
	}
}

func TestConfigure(t *testing.T) {
	Configure(true)
	if !Secure {
		t.Fatal("expected Secure to be true after Configure(true)")
	}
	Configure(false)
	if Secure {
		t.Fatal("expected Secure to be false after Configure(false)")
	}
}
