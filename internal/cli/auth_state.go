package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/myyra/procountor/procountorapi"
)

const (
	authStateFileMode      = 0o600
	authStateDirectoryMode = 0o700
	refreshTokenSkew       = 30 * time.Second
	defaultTokenLifetime   = 40 * time.Minute
)

type authState struct {
	CompanyID    int       `json:"company_id"`
	CompanyName  string    `json:"company_name,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

func (r *runner) resolveAccessToken(ctx context.Context) (string, error) {
	path, err := r.authFilePath()
	if err != nil {
		return "", err
	}

	state, err := loadAuthState(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	now := time.Now().UTC()
	if strings.TrimSpace(state.AccessToken) != "" && state.ExpiresAt.After(now.Add(refreshTokenSkew)) {
		return state.AccessToken, nil
	}

	refreshToken := strings.TrimSpace(state.RefreshToken)
	if refreshToken == "" {
		return "", invalidUsage("stored auth state has no refresh token, run 'procountor auth login'")
	}

	client, err := r.unauthClient(false)
	if err != nil {
		return "", wrapErr("create API client", err)
	}

	res, err := client.CreateToken(ctx, &procountorapi.TokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return "", wrapErr("refresh access token", err)
	}

	state.AccessToken = strings.TrimSpace(res.AccessToken)
	if state.AccessToken == "" {
		return "", errors.New("token endpoint returned empty access_token")
	}
	state.ExpiresAt = tokenExpiresAt(res.ExpiresIn)
	state.UpdatedAt = time.Now().UTC()
	if err := saveAuthState(path, state); err != nil {
		return "", err
	}

	return state.AccessToken, nil
}

func tokenExpiresAt(expiresIn int) time.Time {
	if expiresIn <= 0 {
		expiresIn = int(defaultTokenLifetime.Seconds())
	}
	return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
}

func (r *runner) authFilePath() (string, error) {
	if path := strings.TrimSpace(r.authFile); path != "" {
		return expandHomePath(path)
	}
	return defaultAuthFilePath()
}

func defaultAuthFilePath() (string, error) {
	if xdgStateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "procountor", "auth.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "procountor", "auth.json"), nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home dir: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func loadAuthState(path string) (*authState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read auth state: %w", err)
	}

	var state authState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode auth state: %w", err)
	}
	return &state, nil
}

func saveAuthState(path string, state *authState) error {
	if err := os.MkdirAll(filepath.Dir(path), authStateDirectoryMode); err != nil {
		return fmt.Errorf("create auth directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auth state: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, append(data, '\n'), authStateFileMode); err != nil {
		return fmt.Errorf("write auth state temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace auth state: %w", err)
	}
	return nil
}

func deleteAuthState(path string) (bool, error) {
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("delete auth state: %w", err)
	}
	return true, nil
}
