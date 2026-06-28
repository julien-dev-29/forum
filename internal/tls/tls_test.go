package tls

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTLSConfig_DevMode(t *testing.T) {
	cfg := NewTLSConfig(nil, true)
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected TLS 1.2 min, got %d", cfg.MinVersion)
	}
	if !cfg.PreferServerCipherSuites {
		t.Fatal("expected PreferServerCipherSuites to be true")
	}
	if len(cfg.CipherSuites) == 0 {
		t.Fatal("expected non-empty cipher suites")
	}
}

func TestNewTLSConfig_HTTPMode(t *testing.T) {
	cfg := NewTLSConfig(nil, false)
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	resp := rr.Result()

	checks := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
		"Content-Security-Policy":   "",
	}

	for header, expectedPrefix := range checks {
		value := resp.Header.Get(header)
		if value == "" {
			t.Errorf("missing header: %s", header)
			continue
		}
		if expectedPrefix != "" && value != expectedPrefix {
			t.Errorf("%s: expected %q, got %q", header, expectedPrefix, value)
		}
	}
}

func TestHSTS(t *testing.T) {
	handler := HSTS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	hsts := rr.Result().Header.Get("Strict-Transport-Security")
	if hsts == "" {
		t.Fatal("missing HSTS header")
	}
	if hsts != "max-age=31536000; includeSubDomains" {
		t.Fatalf("unexpected HSTS value: %q", hsts)
	}
}

func TestSecurityHeaders_NotOverrideExisting(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "custom value")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().Header.Get("X-Custom") != "custom value" {
		t.Fatal("handler should be able to set custom headers")
	}
}

func TestHTTPSRedirect_HTTP(t *testing.T) {
	handler := HTTPSRedirect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test-path", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc != "https://example.com/test-path" {
		t.Fatalf("expected redirect to https://example.com/test-path, got %q", loc)
	}
}

func TestHTTPSRedirect_HTTPS(t *testing.T) {
	handler := HTTPSRedirect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for HTTPS request, got %d", rr.Code)
	}
}

func TestHTTPSRedirect_XForwardedProto(t *testing.T) {
	handler := HTTPSRedirect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with X-Forwarded-Proto: https, got %d", rr.Code)
	}
}
