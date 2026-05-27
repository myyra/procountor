package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myyra/procountor/internal/cli"
)

const invalidUsageExitCode = 2
const testLedgerReceipt123Path = "/ledgerreceipts/123"

type authStateFixture struct {
	CompanyID    int       `json:"company_id"`
	CompanyName  string    `json:"company_name,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

func writeAuthStateFixture(t *testing.T, path string, state authStateFixture) {
	t.Helper()

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal auth state fixture: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write auth state fixture: %v", err)
	}
}

func runCLI(t *testing.T, args []string) (int, string, string) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := cli.Execute(context.Background(), args, strings.NewReader(""), &out, &errOut)
	return code, out.String(), errOut.String()
}

func runCLIJSON(t *testing.T, args []string) (int, string, string) {
	t.Helper()

	withJSON := append([]string{"--output", "json"}, args...)
	return runCLI(t, withJSON)
}

func writeAuthStateTempFile(t *testing.T, state authStateFixture) string {
	t.Helper()

	authFile := filepath.Join(t.TempDir(), "auth.json")
	writeAuthStateFixture(t, authFile, state)
	return authFile
}

func writeValidAuthState(t *testing.T, accessToken string) string {
	t.Helper()

	return writeAuthStateTempFile(t, authStateFixture{
		CompanyID:    1,
		AccessToken:  accessToken,
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
		UpdatedAt:    time.Now().UTC(),
	})
}

func assertCommandSucceeded(t *testing.T, code int, stderr string) {
	t.Helper()

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func assertInvalidUsageError(t *testing.T, code int, stdout, stderr, wantMessageSubstring string) {
	t.Helper()

	if code != invalidUsageExitCode {
		t.Fatalf("exit code = %d, want %d", code, invalidUsageExitCode)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	message := strings.TrimSpace(stderr)
	if !strings.Contains(message, wantMessageSubstring) {
		t.Fatalf("message = %q, want substring %q", message, wantMessageSubstring)
	}
}

func TestExecuteUsageErrors(t *testing.T) {
	t.Parallel()

	authFile := filepath.Join(t.TempDir(), "auth.json")

	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{
			name:        "extra arg on no-args command",
			args:        []string{"--auth-file", authFile, "auth", "status", "extra"},
			wantMessage: "unexpected argument extra",
		},
		{
			name:        "invalid output format",
			args:        []string{"--auth-file", authFile, "--output", "yaml", "auth", "status"},
			wantMessage: "--output must be one of",
		},
		{
			name:        "missing interactive username in non-interactive mode",
			args:        []string{"--auth-file", authFile, "auth", "companies"},
			wantMessage: "pass --username or set PROCOUNTOR_USERNAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLI(t, tt.args)
			assertInvalidUsageError(t, code, stdout, stderr, tt.wantMessage)
		})
	}
}

func TestExecuteWithoutArgsShowsRootHelp(t *testing.T) {
	t.Parallel()

	bareCode, bareStdout, bareStderr := runCLI(t, nil)
	helpCode, helpStdout, helpStderr := runCLI(t, []string{"--help"})

	assertCommandSucceeded(t, bareCode, bareStderr)
	assertCommandSucceeded(t, helpCode, helpStderr)

	if bareStdout != helpStdout {
		t.Fatalf("stdout mismatch for bare invocation\nbare:\n%s\nhelp:\n%s", bareStdout, helpStdout)
	}
	if !strings.Contains(bareStdout, "Usage:") {
		t.Fatalf("stdout = %q, want help output", bareStdout)
	}
}

func TestExecuteAuthCompaniesUsesCredentialEnvVars(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "auth.json")
	t.Setenv("PROCOUNTOR_USERNAME", "env-user")
	t.Setenv("PROCOUNTOR_PASSWORD", "env-pass")

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/login/companies" {
			t.Fatalf("path = %s, want /login/companies", r.URL.Path)
		}

		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode login companies request: %v", err)
		}
		if body.Username != "env-user" {
			t.Fatalf("username = %q, want %q", body.Username, "env-user")
		}
		if body.Password != "env-pass" {
			t.Fatalf("password = %q, want %q", body.Password, "env-pass")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":123,"name":"Acme Oy"}]`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"auth", "companies",
	})
	assertCommandSucceeded(t, code, stderr)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}

	var output struct {
		Companies []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"companies"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &output); err != nil {
		t.Fatalf("decode companies output: %v\noutput: %q", err, stdout)
	}
	if len(output.Companies) != 1 || output.Companies[0].ID != 123 || output.Companies[0].Name != "Acme Oy" {
		t.Fatalf("unexpected companies output: %s", stdout)
	}
}

func TestExecuteAuthStatusWhenNotLoggedIn(t *testing.T) {
	t.Parallel()

	authFile := filepath.Join(t.TempDir(), "missing-auth.json")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "auth", "status"})
	assertCommandSucceeded(t, code, stderr)

	want := fmt.Sprintf("logged_in: false\nauth_file: %s\n", authFile)
	if stdout != want {
		t.Fatalf("text output = %q, want %q", stdout, want)
	}
}

func TestExecuteAuthStatusTextOutput(t *testing.T) {
	t.Parallel()

	authFile := filepath.Join(t.TempDir(), "missing-auth.json")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "--output", "human", "auth", "status"})
	assertCommandSucceeded(t, code, stderr)

	want := fmt.Sprintf("logged_in: false\nauth_file: %s\n", authFile)
	if stdout != want {
		t.Fatalf("text output = %q, want %q", stdout, want)
	}
}

func TestExecuteAuthStatusWhenLoggedIn(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().UTC().Add(time.Hour)
	updatedAt := time.Date(2026, time.February, 17, 10, 0, 0, 0, time.UTC)
	authFile := writeAuthStateTempFile(t, authStateFixture{
		CompanyID:    123,
		CompanyName:  "Acme Oy",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    expiresAt,
		UpdatedAt:    updatedAt,
	})

	code, stdout, stderr := runCLIJSON(t, []string{"--auth-file", authFile, "auth", "status"})
	assertCommandSucceeded(t, code, stderr)

	type statusOutput struct {
		LoggedIn            bool   `json:"logged_in"`
		AuthFile            string `json:"auth_file"`
		CompanyID           int    `json:"company_id"`
		Company             string `json:"company"`
		AccessTokenPresent  bool   `json:"access_token_present"`
		RefreshTokenPresent bool   `json:"refresh_token_present"`
		Expired             bool   `json:"expired"`
	}

	var got statusOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("decode status output: %v\noutput: %q", err, stdout)
	}

	if !got.LoggedIn {
		t.Fatalf("logged_in = false, want true")
	}
	if got.AuthFile != authFile {
		t.Fatalf("auth_file = %q, want %q", got.AuthFile, authFile)
	}
	if got.CompanyID != 123 {
		t.Fatalf("company_id = %d, want 123", got.CompanyID)
	}
	if got.Company != "Acme Oy" {
		t.Fatalf("company = %q, want %q", got.Company, "Acme Oy")
	}
	if !got.AccessTokenPresent {
		t.Fatalf("access_token_present = false, want true")
	}
	if !got.RefreshTokenPresent {
		t.Fatalf("refresh_token_present = false, want true")
	}
	if got.Expired {
		t.Fatalf("expired = true, want false")
	}
}

func TestExecuteAuthLogout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		createState bool
		wantRemoved bool
	}{
		{
			name:        "removes existing auth state",
			createState: true,
			wantRemoved: true,
		},
		{
			name:        "returns removed false when no auth state exists",
			createState: false,
			wantRemoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authFile := filepath.Join(t.TempDir(), "auth.json")
			if tt.createState {
				authFile = writeValidAuthState(t, "access-token")
			}

			code, stdout, stderr := runCLIJSON(t, []string{"--auth-file", authFile, "auth", "logout"})
			assertCommandSucceeded(t, code, stderr)

			var out struct {
				OK       bool   `json:"ok"`
				Removed  bool   `json:"removed"`
				AuthFile string `json:"auth_file"`
			}
			if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
				t.Fatalf("decode logout output: %v\noutput: %q", err, stdout)
			}

			if !out.OK {
				t.Fatalf("ok = false, want true")
			}
			if out.Removed != tt.wantRemoved {
				t.Fatalf("removed = %v, want %v", out.Removed, tt.wantRemoved)
			}
			if out.AuthFile != authFile {
				t.Fatalf("auth_file = %q, want %q", out.AuthFile, authFile)
			}
		})
	}
}

func TestExecuteInvoicesSaveRequiresInvoiceImport(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "invoices", "save"})
	assertInvalidUsageError(t, code, stdout, stderr, "--invoice")
}

func TestExecuteLedgerReceiptsUpdateRequiresLedgerReceiptImport(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "ledger-receipts", "update", "123"})
	assertInvalidUsageError(t, code, stdout, stderr, "--ledger-receipt")
}

func TestExecuteLedgerReceiptsGetAcceptsNullableInvoiceIDAndMissingVATPercent(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != testLedgerReceipt123Path {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id": 123,
			"type": "JOURNAL",
			"status": "UNFINISHED",
			"name": "Manual adjustment",
			"receiptDate": "2026-02-01",
			"vatType": "SALES",
			"vatStatus": 1,
			"invoiceId": null,
			"transactions": [
				{"id": 678, "transactionType": "ENTRY", "account": "4000", "accountingValue": 100},
				{"id": 679, "transactionType": "ENTRY", "account": "1910", "accountingValue": -100, "vatPercent": 0}
			]
		}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"ledger-receipts", "get", "123",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		InvoiceID    *int `json:"invoiceId"`
		Transactions []struct {
			VatPercent *float64 `json:"vatPercent"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode output: %v\noutput: %q", err, stdout)
	}
	if out.InvoiceID != nil {
		t.Fatalf("invoiceId = %v, want null", *out.InvoiceID)
	}
	if len(out.Transactions) != 2 {
		t.Fatalf("transactions length = %d, want 2", len(out.Transactions))
	}
	if out.Transactions[0].VatPercent != nil {
		t.Fatalf("first transaction vatPercent = %v, want omitted", *out.Transactions[0].VatPercent)
	}
	if out.Transactions[1].VatPercent == nil || *out.Transactions[1].VatPercent != 0 {
		t.Fatalf("second transaction vatPercent = %v, want 0", out.Transactions[1].VatPercent)
	}
}

func TestExecuteBusinessPartnersPatchRequiresBusinessPartnerImport(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "business-partners", "patch", "123"})
	assertInvalidUsageError(t, code, stdout, stderr, "--business-partner")
}

func TestExecutePersonsUpdateRequiresPersonImport(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "persons", "update", "123"})
	assertInvalidUsageError(t, code, stdout, stderr, "--person")
}

func TestExecuteLedgerReceiptTransactionsUpdateRoundTripsReceipt(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	var getCalls int32
	var putCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testLedgerReceipt123Path:
			atomic.AddInt32(&getCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id": 123,
				"type": "JOURNAL",
				"status": "UNFINISHED",
				"name": "Manual adjustment",
				"receiptDate": "2026-02-01",
				"vatType": "SALES",
				"vatStatus": 1,
				"version": "2026-02-01T12:00:00",
				"transactions": [
					{"id": 678, "transactionType": "ENTRY", "account": "4000", "accountingValue": 100, "vatPercent": 0, "description": "Old description"},
					{"id": 679, "transactionType": "ENTRY", "account": "1910", "accountingValue": -100, "vatPercent": 0}
				],
				"attachments": []
			}`)
		case r.Method == http.MethodPut && r.URL.Path == testLedgerReceipt123Path:
			atomic.AddInt32(&putCalls, 1)

			var body struct {
				Version      string `json:"version"`
				Transactions []struct {
					ID              int     `json:"id"`
					Account         string  `json:"account"`
					Description     string  `json:"description"`
					AccountingValue float64 `json:"accountingValue"`
				} `json:"transactions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update request: %v", err)
			}

			if body.Version != "2026-02-01T12:00:00" {
				t.Fatalf("version = %q, want %q", body.Version, "2026-02-01T12:00:00")
			}
			if len(body.Transactions) != 2 {
				t.Fatalf("transactions length = %d, want 2", len(body.Transactions))
			}
			if body.Transactions[0].ID != 678 || body.Transactions[0].Account != "3000" {
				t.Fatalf("first transaction = %+v, want id 678 and account 3000", body.Transactions[0])
			}
			if body.Transactions[0].Description != "Retagged account" {
				t.Fatalf("description = %q, want %q", body.Transactions[0].Description, "Retagged account")
			}
			if body.Transactions[1].ID != 679 || body.Transactions[1].Account != "1910" {
				t.Fatalf("second transaction = %+v, want unchanged account 1910", body.Transactions[1])
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":123,"type":"JOURNAL","status":"UNFINISHED","name":"Manual adjustment","receiptDate":"2026-02-01","vatType":"SALES","vatStatus":1,"transactions":[{"id":678,"transactionType":"ENTRY","account":"3000","accountingValue":100,"vatPercent":0,"description":"Retagged account"},{"id":679,"transactionType":"ENTRY","account":"1910","accountingValue":-100,"vatPercent":0}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"ledger-receipts", "transactions", "update",
		"123",
		"678",
		"--account", "3000",
		"--description", "Retagged account",
	})
	assertCommandSucceeded(t, code, stderr)
	if atomic.LoadInt32(&getCalls) != 1 {
		t.Fatalf("GET count = %d, want 1", atomic.LoadInt32(&getCalls))
	}
	if atomic.LoadInt32(&putCalls) != 1 {
		t.Fatalf("PUT count = %d, want 1", atomic.LoadInt32(&putCalls))
	}

	var out struct {
		Transactions []struct {
			ID      int    `json:"id"`
			Account string `json:"account"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode update output: %v\noutput: %q", err, stdout)
	}
	if len(out.Transactions) != 2 || out.Transactions[0].Account != "3000" {
		t.Fatalf("unexpected output transactions: %s", stdout)
	}
}

func TestExecuteLedgerReceiptTransactionsAddRoundTripsReceipt(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testLedgerReceipt123Path:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id": 123,
				"type": "JOURNAL",
				"status": "UNFINISHED",
				"name": "Manual adjustment",
				"receiptDate": "2026-02-01",
				"vatType": "SALES",
				"vatStatus": 1,
				"version": "2026-02-01T12:00:00",
				"transactions": [
					{"id": 678, "transactionType": "ENTRY", "account": "1910", "accountingValue": -100, "vatPercent": 0}
				],
				"attachments": []
			}`)
		case r.Method == http.MethodPut && r.URL.Path == testLedgerReceipt123Path:
			var body struct {
				Transactions []struct {
					Account         string  `json:"account"`
					AccountingValue float64 `json:"accountingValue"`
					TransactionType string  `json:"transactionType"`
					VatPercent      float64 `json:"vatPercent"`
				} `json:"transactions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode add request: %v", err)
			}
			if len(body.Transactions) != 2 {
				t.Fatalf("transactions length = %d, want 2", len(body.Transactions))
			}
			added := body.Transactions[1]
			if added.TransactionType != "ENTRY" || added.Account != "3000" || added.AccountingValue != 100 || added.VatPercent != 0 {
				t.Fatalf("added transaction = %+v, want new entry", added)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":123,"type":"JOURNAL","status":"UNFINISHED","name":"Manual adjustment","receiptDate":"2026-02-01","vatType":"SALES","vatStatus":1,"transactions":[{"id":678,"transactionType":"ENTRY","account":"1910","accountingValue":-100,"vatPercent":0},{"transactionType":"ENTRY","account":"3000","accountingValue":100,"vatPercent":0}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	code, _, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"ledger-receipts", "transactions", "add",
		"123",
		"--transaction-type", "ENTRY",
		"--account", "3000",
		"--accounting-value", "100",
		"--vat-percent", "0",
	})
	assertCommandSucceeded(t, code, stderr)
}

func TestExecuteLedgerReceiptTransactionsRemoveRoundTripsReceipt(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testLedgerReceipt123Path:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id": 123,
				"type": "JOURNAL",
				"status": "UNFINISHED",
				"name": "Manual adjustment",
				"receiptDate": "2026-02-01",
				"vatType": "SALES",
				"vatStatus": 1,
				"version": "2026-02-01T12:00:00",
				"transactions": [
					{"id": 678, "transactionType": "ENTRY", "account": "3000", "accountingValue": 100, "vatPercent": 0},
					{"id": 679, "transactionType": "ENTRY", "account": "1910", "accountingValue": -100, "vatPercent": 0}
				],
				"attachments": []
			}`)
		case r.Method == http.MethodPut && r.URL.Path == testLedgerReceipt123Path:
			var body struct {
				Transactions []struct {
					ID int `json:"id"`
				} `json:"transactions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode remove request: %v", err)
			}
			if len(body.Transactions) != 1 || body.Transactions[0].ID != 679 {
				t.Fatalf("transactions = %+v, want only id 679", body.Transactions)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":123,"type":"JOURNAL","status":"UNFINISHED","name":"Manual adjustment","receiptDate":"2026-02-01","vatType":"SALES","vatStatus":1,"transactions":[{"id":679,"transactionType":"ENTRY","account":"1910","accountingValue":-100,"vatPercent":0}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	code, _, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"ledger-receipts", "transactions", "remove",
		"123",
		"678",
	})
	assertCommandSucceeded(t, code, stderr)
}

func TestExecuteVatsCountryRequiresCountryCode(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")
	code, stdout, stderr := runCLI(t, []string{"--auth-file", authFile, "vats", "country"})
	assertInvalidUsageError(t, code, stdout, stderr, "country-code")
}

func TestExecuteVatsCountry(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/vats/country" {
			t.Fatalf("path = %s, want /vats/country", r.URL.Path)
		}
		if got := r.URL.Query().Get("countryCode"); got != "DK" {
			t.Fatalf("countryCode = %q, want %q", got, "DK")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vatPercentages":[25,12]}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"vats", "country",
		"--country-code", "DK",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		VatPercentages []float64 `json:"vatPercentages"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode VAT country output: %v\noutput: %q", err, stdout)
	}
	if len(out.VatPercentages) != 2 || out.VatPercentages[0] != 25 || out.VatPercentages[1] != 12 {
		t.Fatalf("unexpected VAT country output: %s", stdout)
	}
}

func TestExecuteVatsCompany(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/vats/default" {
			t.Fatalf("path = %s, want /vats/default", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vatInformation":[{"country":"Finland","percentages":[{"vatPercent":25.5,"sales":true,"purchase":true}]}],"vatStatuses":[{"vatStatus":1,"description":"vat_25_5","sales":true,"purchase":true}]}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"vats", "company",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		VatInformation []struct {
			Country string `json:"country"`
		} `json:"vatInformation"`
		VatStatuses []struct {
			VatStatus   int    `json:"vatStatus"`
			Description string `json:"description"`
		} `json:"vatStatuses"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode VAT company output: %v\noutput: %q", err, stdout)
	}
	if len(out.VatInformation) != 1 || out.VatInformation[0].Country != "Finland" {
		t.Fatalf("unexpected VAT company information: %s", stdout)
	}
	if len(out.VatStatuses) != 1 || out.VatStatuses[0].VatStatus != 1 {
		t.Fatalf("unexpected VAT company statuses: %s", stdout)
	}
}

func TestExecuteVatsSettings(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/vats/settings" {
			t.Fatalf("path = %s, want /vats/settings", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"unitPricesIncludeVatSales":true,"unitPricesIncludeVatPurchase":false}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"vats", "settings",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		UnitPricesIncludeVatSales    bool `json:"unitPricesIncludeVatSales"`
		UnitPricesIncludeVatPurchase bool `json:"unitPricesIncludeVatPurchase"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode VAT settings output: %v\noutput: %q", err, stdout)
	}
	if !out.UnitPricesIncludeVatSales || out.UnitPricesIncludeVatPurchase {
		t.Fatalf("unexpected VAT settings output: %s", stdout)
	}
}

func TestExecuteInvoiceCommentsCreateUsesEmbeddedTaggedUsersFlags(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/invoices/123/comments" {
			t.Fatalf("path = %s, want /invoices/123/comments", r.URL.Path)
		}

		var body struct {
			Comment         string `json:"comment"`
			TaggedUsersInfo struct {
				NumTaggedUsers       int  `json:"numTaggedUsers"`
				ReadByAllTaggedUsers bool `json:"readByAllTaggedUsers"`
			} `json:"taggedUsersInfo"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode comment request: %v", err)
		}
		if body.Comment != "hello" {
			t.Fatalf("comment = %q, want hello", body.Comment)
		}
		if body.TaggedUsersInfo.NumTaggedUsers != 2 {
			t.Fatalf("numTaggedUsers = %d, want 2", body.TaggedUsersInfo.NumTaggedUsers)
		}
		if !body.TaggedUsersInfo.ReadByAllTaggedUsers {
			t.Fatal("readByAllTaggedUsers = false, want true")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"comment":"hello","taggedUsersInfo":{"numTaggedUsers":2,"readByAllTaggedUsers":true}}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"invoices", "comments", "create",
		"123",
		"--comment", "hello",
		"--tagged-users-info.num-tagged-users", "2",
		"--tagged-users-info.read-by-all-tagged-users", "true",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		Comment         string `json:"comment"`
		TaggedUsersInfo struct {
			NumTaggedUsers       int  `json:"numTaggedUsers"`
			ReadByAllTaggedUsers bool `json:"readByAllTaggedUsers"`
		} `json:"taggedUsersInfo"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode invoice comment output: %v\noutput: %q", err, stdout)
	}
	if out.Comment != "hello" || out.TaggedUsersInfo.NumTaggedUsers != 2 || !out.TaggedUsersInfo.ReadByAllTaggedUsers {
		t.Fatalf("unexpected invoice comment output: %s", stdout)
	}
}

func TestExecuteReportsAccountingUsesExplicitFlags(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/reports/accounting" {
			t.Fatalf("path = %s, want /reports/accounting", r.URL.Path)
		}

		var body struct {
			EndDate string `json:"endDate"`
			Type    string `json:"type"`
			Options struct {
				ReceiptName string `json:"receiptName"`
			} `json:"options"`
			ReceiptStatus []string `json:"receiptStatus"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode accounting report request: %v", err)
		}
		if body.EndDate != "2026-03-31" {
			t.Fatalf("endDate = %q, want 2026-03-31", body.EndDate)
		}
		if body.Type != "BALANCE_SHEET" {
			t.Fatalf("type = %q, want BALANCE_SHEET", body.Type)
		}
		if body.Options.ReceiptName != "Report" {
			t.Fatalf("receiptName = %q, want Report", body.Options.ReceiptName)
		}
		if len(body.ReceiptStatus) != 1 || body.ReceiptStatus[0] != "APPROVED" {
			t.Fatalf("receiptStatus = %+v, want [APPROVED]", body.ReceiptStatus)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"reports", "accounting",
		"--end-date", "2026-03-31",
		"--type", "BALANCE_SHEET",
		"--receipt-status", "APPROVED",
		"--options.receipt-name", "Report",
	})
	assertCommandSucceeded(t, code, stderr)
	if strings.TrimSpace(stdout) != "{}" {
		t.Fatalf("stdout = %q, want {}", strings.TrimSpace(stdout))
	}
}

func TestExecuteBusinessPartnerGroupsCreateUsesExplicitFlags(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/businesspartners/groups" {
			t.Fatalf("path = %s, want /businesspartners/groups", r.URL.Path)
		}

		var body struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Active bool   `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode business partner group request: %v", err)
		}
		if body.Name != "VIP" || body.Type != "CUSTOMER" || !body.Active {
			t.Fatalf("unexpected request body: %+v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":123,"name":"VIP","type":"CUSTOMER","active":true}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"business-partners", "groups", "create",
		"--name", "VIP",
		"--type", "CUSTOMER",
		"--active", "true",
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Type   string `json:"type"`
		Active bool   `json:"active"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode business partner group output: %v\noutput: %q", err, stdout)
	}
	if out.ID != 123 || out.Name != "VIP" || out.Type != "CUSTOMER" || !out.Active {
		t.Fatalf("unexpected business partner group output: %s", stdout)
	}
}

func TestExecutePersonsUpdateUsesPersonImport(t *testing.T) {
	t.Parallel()

	authFile := writeValidAuthState(t, "access-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/persons/123" {
			t.Fatalf("path = %s, want /persons/123", r.URL.Path)
		}

		var body struct {
			FirstName       string `json:"firstName"`
			LastName        string `json:"lastName"`
			JobTitle        string `json:"jobTitle"`
			Version         string `json:"version"`
			DeliveryAddress struct {
				Street string `json:"street"`
				City   string `json:"city"`
			} `json:"deliveryAddress"`
			PaymentInfo struct {
				PaymentMethod     string  `json:"paymentMethod"`
				PaymentTermDays   string  `json:"paymentTermDays"`
				PenalInterestRate float64 `json:"penalInterestRate"`
			} `json:"paymentInfo"`
			InvoicingInfo struct {
				Language string `json:"language"`
				Gender   string `json:"gender"`
			} `json:"invoicingInfo"`
			RegistryInfo struct {
				Active         bool  `json:"active"`
				PersonGroupsID []int `json:"personGroupsIds"`
			} `json:"registryInfo"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode person request: %v", err)
		}
		if body.FirstName != "Ada" || body.LastName != "Lovelace" || body.JobTitle != "Consultant" {
			t.Fatalf("unexpected top-level body: %+v", body)
		}
		if body.DeliveryAddress.Street != "Example 1" || body.DeliveryAddress.City != "Helsinki" {
			t.Fatalf("unexpected delivery address: %+v", body.DeliveryAddress)
		}
		if body.PaymentInfo.PaymentMethod != "BANK_TRANSFER" || body.PaymentInfo.PaymentTermDays != "14" || body.PaymentInfo.PenalInterestRate != 7.5 {
			t.Fatalf("unexpected payment info: %+v", body.PaymentInfo)
		}
		if body.InvoicingInfo.Language != "ENGLISH" || body.InvoicingInfo.Gender != "FEMALE" {
			t.Fatalf("unexpected invoicing info: %+v", body.InvoicingInfo)
		}
		if !body.RegistryInfo.Active || len(body.RegistryInfo.PersonGroupsID) != 2 || body.RegistryInfo.PersonGroupsID[0] != 10 || body.RegistryInfo.PersonGroupsID[1] != 20 {
			t.Fatalf("unexpected registry info: %+v", body.RegistryInfo)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":123,
			"firstName":"Ada",
			"lastName":"Lovelace",
			"jobTitle":"Consultant",
			"paymentInfo":{"paymentMethod":"BANK_TRANSFER","paymentTermDays":"14","penalInterestRate":7.5},
			"invoicingInfo":{"language":"ENGLISH","gender":"FEMALE"},
			"registryInfo":{"active":true,"personGroupsIds":[10,20]},
			"version":"2026-03-11T12:00:00"
		}`)
	}))
	defer server.Close()

	code, stdout, stderr := runCLIJSON(t, []string{
		"--base-url", server.URL,
		"--auth-file", authFile,
		"persons", "update",
		"123",
		"--person", `{
			"firstName":"Ada",
			"lastName":"Lovelace",
			"jobTitle":"Consultant",
			"deliveryAddress":{"street":"Example 1","city":"Helsinki"},
			"paymentInfo":{"paymentMethod":"BANK_TRANSFER","paymentTermDays":"14","penalInterestRate":7.5},
			"invoicingInfo":{"language":"ENGLISH","gender":"FEMALE"},
			"registryInfo":{"active":true,"personGroupsIds":[10,20]},
			"version":"2026-03-11T12:00:00"
		}`,
	})
	assertCommandSucceeded(t, code, stderr)

	var out struct {
		ID        int    `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		JobTitle  string `json:"jobTitle"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &out); err != nil {
		t.Fatalf("decode person output: %v\noutput: %q", err, stdout)
	}
	if out.ID != 123 || out.FirstName != "Ada" || out.LastName != "Lovelace" || out.JobTitle != "Consultant" {
		t.Fatalf("unexpected person output: %s", stdout)
	}
}
