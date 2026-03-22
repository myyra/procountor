package cli_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDimensionsListPassesQueryAndAuth(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "list-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/dimensions" {
			t.Fatalf("path = %s, want /dimensions", r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "Sales" {
			t.Fatalf("query name = %q, want %q", got, "Sales")
		}
		if got := r.URL.Query().Get("codeName"); got != "Internal" {
			t.Fatalf("query codeName = %q, want %q", got, "Internal")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer list-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer list-token")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"name":"Sales","items":[]}]`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"dimensions", "list",
		"--name", "Sales",
		"--code-name", "Internal",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}

	var dimensions []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &dimensions); err != nil {
		t.Fatalf("decode stdout: %v", err)
	}
	if len(dimensions) != 1 || dimensions[0].Name != "Sales" {
		t.Fatalf("unexpected dimensions payload: %s", stdout)
	}
}

func TestDimensionsUpdatePassesPathBodyAndAuth(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "update-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/dimensions/42" {
			t.Fatalf("path = %s, want /dimensions/42", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer update-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer update-token")
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Name != "Renamed Dimension" {
			t.Fatalf("body.name = %q, want %q", body.Name, "Renamed Dimension")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"Renamed Dimension"}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"dimensions", "update",
		"42",
		"--name", "Renamed Dimension",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}

	var response struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &response); err != nil {
		t.Fatalf("decode stdout: %v", err)
	}
	if response.Name != "Renamed Dimension" {
		t.Fatalf("response name = %q, want %q", response.Name, "Renamed Dimension")
	}
}

func TestDimensionsUpdateMissingDimensionIDIsInvalidUsage(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "update-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.NotFound(w, r)
	}))
	defer server.Close()

	code, stdout, stderr := runCLI(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"dimensions", "update",
		"--name", "Renamed Dimension",
	})
	assertInvalidUsageError(t, code, stdout, stderr, "<dimension-id>")
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
}
