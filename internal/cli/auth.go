package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/myyra/procountor/procountorapi"
)

type AuthCmd struct {
	Companies AuthCompaniesCmd `cmd:"" help:"List companies you can access with your credentials."`
	Login     AuthLoginCmd     `cmd:"" help:"Sign in and save session tokens to the auth file."`
	Refresh   AuthRefreshCmd   `cmd:"" help:"Refresh the stored access token."`
	Status    AuthStatusCmd    `cmd:"" help:"Show whether local session state is usable."`
	Logout    AuthLogoutCmd    `cmd:"" help:"Delete local session state."`
}

type AuthCompaniesCmd struct {
	Username string `help:"Procountor username. Falls back to PROCOUNTOR_USERNAME and prompts if omitted." name:"username"`
	Password string `help:"Procountor password. Falls back to PROCOUNTOR_PASSWORD and prompts if omitted." name:"password"`
}

func (c *AuthCompaniesCmd) Run(ctx context.Context, r *runner) error {
	creds, err := r.credentialsFromInput(c.Username, c.Password)
	if err != nil {
		return err
	}

	client, err := r.unauthClient(false)
	if err != nil {
		return wrapErr("create API client", err)
	}

	res, err := client.GetLoginCompanies(ctx, &procountorapi.LoginCompaniesRequest{
		Username: creds.username,
		Password: creds.password,
	})
	if err != nil {
		return wrapErr("get login companies", err)
	}
	return r.writeOutput(map[string]any{"companies": res.Response})
}

type AuthLoginCmd struct {
	CompanyID *int   `help:"Company ID to log into. Defaults to the first accessible company."              name:"company-id"`
	Username  string `help:"Procountor username. Falls back to PROCOUNTOR_USERNAME and prompts if omitted." name:"username"`
	Password  string `help:"Procountor password. Falls back to PROCOUNTOR_PASSWORD and prompts if omitted." name:"password"`
}

func (c *AuthLoginCmd) Run(ctx context.Context, r *runner) error {
	creds, err := r.credentialsFromInput(c.Username, c.Password)
	if err != nil {
		return err
	}

	client, err := r.unauthClient(true)
	if err != nil {
		return wrapErr("create API client", err)
	}

	companiesRes, err := client.GetLoginCompanies(ctx, &procountorapi.LoginCompaniesRequest{
		Username: creds.username,
		Password: creds.password,
	})
	if err != nil {
		return wrapErr("get login companies", err)
	}
	if len(companiesRes.Response) == 0 {
		return errors.New("no accessible companies found for these credentials")
	}

	requestedID := 0
	hasRequestedID := c.CompanyID != nil
	if c.CompanyID != nil {
		requestedID = *c.CompanyID
	}
	selectedCompanyID, selectedCompanyName, err := chooseCompany(companiesRes.Response, requestedID, hasRequestedID)
	if err != nil {
		return err
	}

	loginRes, err := client.Login(ctx, &procountorapi.LoginRequest{
		Username:   creds.username,
		Password:   creds.password,
		CompanyIds: []string{strconv.Itoa(selectedCompanyID)},
	})
	if err != nil {
		return wrapErr("login", err)
	}

	companyToken, ok := findCompanyToken(loginRes.CompanyTokens, selectedCompanyID)
	if !ok {
		return fmt.Errorf("login response did not include token for company %d", selectedCompanyID)
	}

	path, err := r.authFilePath()
	if err != nil {
		return err
	}

	expiresAt := tokenExpiresAt(companyToken.ApiTokens.ExpiresIn)
	state := &authState{
		CompanyID:    selectedCompanyID,
		CompanyName:  selectedCompanyName,
		AccessToken:  strings.TrimSpace(companyToken.ApiTokens.AccessToken),
		RefreshToken: strings.TrimSpace(companyToken.ApiTokens.RefreshToken),
		ExpiresAt:    expiresAt,
		UpdatedAt:    time.Now().UTC(),
	}
	if state.AccessToken == "" {
		return errors.New("login response did not include access_token")
	}
	if state.RefreshToken == "" {
		return errors.New("login response did not include refresh_token")
	}
	if err := saveAuthState(path, state); err != nil {
		return err
	}

	return r.writeOutput(map[string]any{
		"ok":         true,
		"auth_file":  path,
		"company_id": selectedCompanyID,
		"company":    selectedCompanyName,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func chooseCompany(companies []procountorapi.LoginCompany, requestedID int, hasRequestedID bool) (int, string, error) {
	if !hasRequestedID {
		return companies[0].ID, companies[0].Name, nil
	}
	for _, company := range companies {
		if company.ID == requestedID {
			return company.ID, company.Name, nil
		}
	}
	return 0, "", invalidUsage("company-id %d is not accessible for this user", requestedID)
}

func findCompanyToken(tokens []procountorapi.LoginCompanyToken, companyID int) (procountorapi.LoginCompanyToken, bool) {
	for _, token := range tokens {
		if token.CompanyId == companyID {
			return token, true
		}
	}
	return procountorapi.LoginCompanyToken{}, false
}

type credentials struct {
	username string
	password string
}

func (r *runner) unauthClient(withCookieJar bool) (*procountorapi.Client, error) {
	baseURL := strings.TrimSpace(r.baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient, err := r.newHTTPClient(withCookieJar)
	if err != nil {
		return nil, err
	}

	client, err := procountorapi.NewClient(
		baseURL,
		staticSecurity{token: ""},
		procountorapi.WithClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}
	return client, nil
}

func (r *runner) credentialsFromInput(username, password string) (credentials, error) {
	creds := credentials{
		username: strings.TrimSpace(username),
		password: strings.TrimSpace(password),
	}
	if creds.username == "" {
		creds.username = strings.TrimSpace(os.Getenv(envUsername))
	}
	if creds.password == "" {
		creds.password = strings.TrimSpace(os.Getenv(envPassword))
	}
	if creds.username == "" {
		promptedUsername, err := r.promptValue("Username: ", false)
		if err != nil {
			return credentials{}, err
		}
		creds.username = promptedUsername
	}
	if creds.password == "" {
		promptedPassword, err := r.promptValue("Password: ", true)
		if err != nil {
			return credentials{}, err
		}
		creds.password = promptedPassword
	}
	if creds.username == "" {
		return credentials{}, invalidUsage("username cannot be empty")
	}
	if creds.password == "" {
		return credentials{}, invalidUsage("password cannot be empty")
	}
	return creds, nil
}

func (r *runner) promptValue(prompt string, secret bool) (string, error) {
	if r.inFile == nil || !term.IsTerminal(int(r.inFile.Fd())) {
		if secret {
			return "", invalidUsage("missing password in non-interactive mode: pass --password or set %s", envPassword)
		}
		return "", invalidUsage("missing username in non-interactive mode: pass --username or set %s", envUsername)
	}

	if _, err := io.WriteString(r.errOut, prompt); err != nil {
		return "", wrapErr("write prompt", err)
	}

	if secret {
		bytes, err := term.ReadPassword(int(r.inFile.Fd()))
		if _, writeErr := io.WriteString(r.errOut, "\n"); writeErr != nil {
			return "", wrapErr("write prompt newline", writeErr)
		}
		if err != nil {
			return "", wrapErr("read password", err)
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	line, err := r.lineIn.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", wrapErr("read username", err)
	}
	return strings.TrimSpace(line), nil
}

type AuthRefreshCmd struct {
	Raw bool `help:"Write access token as plain text instead of structured output." name:"raw"`
}

func (c *AuthRefreshCmd) Run(ctx context.Context, r *runner) error {
	path, err := r.authFilePath()
	if err != nil {
		return err
	}

	state, err := loadAuthState(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return invalidUsage("not logged in: run 'procountor auth login'")
		}
		return err
	}
	if strings.TrimSpace(state.RefreshToken) == "" {
		return invalidUsage("stored auth state has no refresh token, run 'procountor auth login'")
	}

	client, err := r.unauthClient(false)
	if err != nil {
		return wrapErr("create API client", err)
	}
	res, err := client.CreateToken(ctx, &procountorapi.TokenRequest{RefreshToken: state.RefreshToken})
	if err != nil {
		return wrapErr("refresh access token", err)
	}

	state.AccessToken = strings.TrimSpace(res.AccessToken)
	state.ExpiresAt = tokenExpiresAt(res.ExpiresIn)
	state.UpdatedAt = time.Now().UTC()
	if err := saveAuthState(path, state); err != nil {
		return err
	}

	if c.Raw {
		if r.mode() != outputModeHuman && r.mode() != outputModePlain {
			return invalidUsage("--raw cannot be combined with --output %s", r.output)
		}
		_, err := io.WriteString(r.out, state.AccessToken+"\n")
		return wrapErr("write token output", err)
	}

	return r.writeOutput(map[string]any{
		"ok":         true,
		"auth_file":  path,
		"company_id": state.CompanyID,
		"expires_at": state.ExpiresAt.Format(time.RFC3339),
	})
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(_ context.Context, r *runner) error {
	path, err := r.authFilePath()
	if err != nil {
		return err
	}

	state, err := loadAuthState(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return r.writeOutput(map[string]any{"logged_in": false, "auth_file": path})
		}
		return err
	}

	now := time.Now().UTC()
	return r.writeOutput(map[string]any{
		"logged_in":             true,
		"auth_file":             path,
		"company_id":            state.CompanyID,
		"company":               state.CompanyName,
		"access_token_present":  strings.TrimSpace(state.AccessToken) != "",
		"refresh_token_present": strings.TrimSpace(state.RefreshToken) != "",
		"expires_at":            state.ExpiresAt.Format(time.RFC3339),
		"expired":               !state.ExpiresAt.After(now.Add(refreshTokenSkew)),
		"updated_at":            state.UpdatedAt.Format(time.RFC3339),
	})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(_ context.Context, r *runner) error {
	path, err := r.authFilePath()
	if err != nil {
		return err
	}
	removed, err := deleteAuthState(path)
	if err != nil {
		return err
	}
	return r.writeOutput(map[string]any{"ok": true, "removed": removed, "auth_file": path})
}
