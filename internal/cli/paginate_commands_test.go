package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBusinessPartnersSearchPaginateLimitFrom(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "partners-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/businesspartners" {
			t.Fatalf("path = %s, want /businesspartners", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer partners-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer partners-token")
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			t.Fatalf("invalid page query: %v", err)
		}
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil {
			t.Fatalf("invalid size query: %v", err)
		}
		if page != 1 {
			t.Fatalf("page = %d, want 1", page)
		}
		if size != 200 {
			t.Fatalf("size = %d, want 200", size)
		}

		if err := writeBusinessPartnerSearchResponse(w, 200, 200); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"business-partners", "search",
		"--paginate", "210:15",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}

	var got []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout: %q", err, stdout)
	}
	if len(got) != 15 {
		t.Fatalf("result count = %d, want 15", len(got))
	}
	if got[0].ID != 210 || got[len(got)-1].ID != 224 {
		t.Fatalf("id range = %d..%d, want 210..224", got[0].ID, got[len(got)-1].ID)
	}
}

func TestBusinessPartnersSearchPaginateAllAcrossPages(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "partners-token")

	var totalCalls int32
	var page1Calls int32
	var page2Calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&totalCalls, 1)

		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/businesspartners" {
			t.Fatalf("path = %s, want /businesspartners", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer partners-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer partners-token")
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			t.Fatalf("invalid page query: %v", err)
		}
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil {
			t.Fatalf("invalid size query: %v", err)
		}
		if size != 200 {
			t.Fatalf("size = %d, want 200", size)
		}

		switch page {
		case 1:
			atomic.AddInt32(&page1Calls, 1)
			if err := writeBusinessPartnerSearchResponse(w, 200, 200); err != nil {
				t.Fatalf("write response: %v", err)
			}
		case 2:
			atomic.AddInt32(&page2Calls, 1)
			if err := writeBusinessPartnerSearchResponse(w, 400, 150); err != nil {
				t.Fatalf("write response: %v", err)
			}
		default:
			t.Fatalf("unexpected page = %d", page)
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"business-partners", "search",
		"--paginate", "395:all",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&totalCalls); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
	if got := atomic.LoadInt32(&page1Calls); got != 1 {
		t.Fatalf("page 1 calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&page2Calls); got != 1 {
		t.Fatalf("page 2 calls = %d, want 1", got)
	}

	var got []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout: %q", err, stdout)
	}
	if len(got) != 155 {
		t.Fatalf("result count = %d, want 155", len(got))
	}
	if got[0].ID != 395 || got[len(got)-1].ID != 549 {
		t.Fatalf("id range = %d..%d, want 395..549", got[0].ID, got[len(got)-1].ID)
	}
}

func TestBusinessPartnersSearchInvalidPaginateIsUsageError(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "partners-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.NotFound(w, r)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"business-partners", "search",
		"--paginate", "abc",
	})
	assertInvalidUsageError(t, code, stdout, stderr, "invalid --paginate")
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
}

func TestCostReceiptsSearchSlicesResultsByPaginate(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "cost-token")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/costreceipt" {
			t.Fatalf("path = %s, want /costreceipt", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cost-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer cost-token")
		}

		if err := writeCostReceiptSearchResponse(w, 0, 5); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"cost-receipts", "search",
		"--paginate", "1:2",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}

	var got []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout: %q", err, stdout)
	}
	if len(got) != 2 {
		t.Fatalf("result count = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("ids = [%d, %d], want [1, 2]", got[0].ID, got[1].ID)
	}
}

func writeBusinessPartnerSearchResponse(w http.ResponseWriter, fromID, count int) error {
	type businessPartner struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	type searchResult struct {
		Results []businessPartner `json:"results"`
	}

	results := make([]businessPartner, 0, count)
	for i := range count {
		id := fromID + i
		results = append(results, businessPartner{
			ID:   id,
			Name: "Partner " + strconv.Itoa(id),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(searchResult{
		Results: results,
	}); err != nil {
		return fmt.Errorf("encode business partner search response: %w", err)
	}
	return nil
}

func writeCostReceiptSearchResponse(w http.ResponseWriter, fromID, count int) error {
	type costReceipt struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Currency    string `json:"currency"`
		ReceiptDate string `json:"receiptDate"`
	}
	type searchResult struct {
		Results []costReceipt `json:"results"`
	}

	results := make([]costReceipt, 0, count)
	for i := range count {
		id := fromID + i
		results = append(results, costReceipt{
			ID:          id,
			Title:       "Receipt " + strconv.Itoa(id),
			Currency:    "EUR",
			ReceiptDate: "2026-02-17",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(searchResult{
		Results: results,
	}); err != nil {
		return fmt.Errorf("encode cost receipt search response: %w", err)
	}
	return nil
}
