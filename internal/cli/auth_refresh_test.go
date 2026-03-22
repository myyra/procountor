package cli_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteRefreshesExpiredStoredToken(t *testing.T) {
	t.Parallel()

	authFile := writeAuthStateTempFile(t, authStateFixture{
		CompanyID:    77,
		AccessToken:  "expired-access-token",
		RefreshToken: "refresh-token-old",
		ExpiresAt:    time.Now().UTC().Add(-5 * time.Minute),
		UpdatedAt:    time.Now().UTC().Add(-1 * time.Hour),
	})

	var tokenCalls int32
	var dimensionsCalls int32
	var gotRefreshToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			atomic.AddInt32(&tokenCalls, 1)
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}

			var body struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode token request body: %v", err)
			}
			gotRefreshToken = body.RefreshToken

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"access_token":"fresh-access-token","expires_in":3600}`)

		case "/dimensions":
			atomic.AddInt32(&dimensionsCalls, 1)
			if got := r.Header.Get("Authorization"); got != "Bearer fresh-access-token" {
				t.Fatalf("Authorization = %q, want %q", got, "Bearer fresh-access-token")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"dimensions", "list",
	})
	assertCommandSucceeded(t, code, stderr)
	if strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("stdout = %q, want JSON array", stdout)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("/token calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&dimensionsCalls); got != 1 {
		t.Fatalf("/dimensions calls = %d, want 1", got)
	}
	if gotRefreshToken != "refresh-token-old" {
		t.Fatalf("refresh token = %q, want %q", gotRefreshToken, "refresh-token-old")
	}

	type persistedState struct {
		AccessToken  string    `json:"access_token"`
		RefreshToken string    `json:"refresh_token"`
		ExpiresAt    time.Time `json:"expires_at"`
	}

	rawState, err := os.ReadFile(authFile)
	if err != nil {
		t.Fatalf("read auth state: %v", err)
	}
	var persisted persistedState
	if err := json.Unmarshal(rawState, &persisted); err != nil {
		t.Fatalf("decode auth state: %v", err)
	}

	if persisted.AccessToken != "fresh-access-token" {
		t.Fatalf("persisted access_token = %q, want %q", persisted.AccessToken, "fresh-access-token")
	}
	if persisted.RefreshToken != "refresh-token-old" {
		t.Fatalf("persisted refresh_token = %q, want %q", persisted.RefreshToken, "refresh-token-old")
	}
	if !persisted.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("persisted expires_at = %s, want future timestamp", persisted.ExpiresAt)
	}
}
