package xbookmarks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	authorizeURL = "https://twitter.com/i/oauth2/authorize"
	tokenURL     = "https://api.twitter.com/2/oauth2/token"
	redirectURI  = "http://127.0.0.1:8787/callback"
	callbackPort = "127.0.0.1:8787"
	scopes       = "tweet.read users.read bookmark.read offline.access"
)

// Token is the persisted OAuth 2.0 token for the X API.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	TokenType    string    `json:"token_type"`
}

// Valid reports whether the token is present and not imminently expiring.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && time.Until(t.ExpiresAt) > 30*time.Second
}

// LoadToken reads the persisted token from disk. Returns (nil, nil) if
// the file is absent, to let callers distinguish "not logged in" from
// "something went wrong."
func LoadToken() (*Token, error) {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return &tok, nil
}

// SaveToken writes the token to disk with 0600 perms.
func SaveToken(tok *Token) error {
	path := TokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ClientID returns the X OAuth client id. PKCE treats the client id as
// public, so we accept it from env (HOTBREW_X_CLIENT_ID) or config — no
// secret handling needed.
func ClientID(fromConfig string) string {
	if v := os.Getenv("HOTBREW_X_CLIENT_ID"); v != "" {
		return v
	}
	return fromConfig
}

// RunPKCEFlow starts the browser-based authorization dance: spin up a
// localhost callback server, open the authorize URL, wait for the code,
// exchange it for tokens, and persist them.
func RunPKCEFlow(ctx context.Context, clientID string) (*Token, error) {
	if clientID == "" {
		return nil, errors.New("x client id not configured (set HOTBREW_X_CLIENT_ID or x.client_id in hotbrew.yaml)")
	}

	verifier, challenge, err := newPKCEPair()
	if err != nil {
		return nil, fmt.Errorf("pkce pair: %w", err)
	}
	state, err := randomString(24)
	if err != nil {
		return nil, fmt.Errorf("state: %w", err)
	}

	authURL := buildAuthorizeURL(clientID, challenge, state)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              callbackPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- errors.New("state mismatch on OAuth callback")
			return
		}
		if errMsg := q.Get("error"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("authorize error: %s", errMsg)
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- errors.New("authorize callback missing code")
			return
		}
		_, _ = fmt.Fprintln(w, "hotbrew: authorization received. You can close this tab.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Println("Opening browser to authorize hotbrew…")
	fmt.Println("If it doesn't open, visit:")
	fmt.Println("  " + authURL)
	_ = openBrowser(authURL)

	select {
	case code := <-codeCh:
		tok, err := exchangeCode(ctx, clientID, code, verifier)
		if err != nil {
			return nil, err
		}
		if err := SaveToken(tok); err != nil {
			return nil, fmt.Errorf("save token: %w", err)
		}
		return tok, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func buildAuthorizeURL(clientID, challenge, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scopes)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return authorizeURL + "?" + q.Encode()
}

func exchangeCode(ctx context.Context, clientID, code, verifier string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	return postToken(ctx, form)
}

// RefreshToken swaps a refresh token for a fresh access token.
func RefreshToken(ctx context.Context, clientID, refresh string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", refresh)
	return postToken(ctx, form)
}

func postToken(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &Token{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second),
		Scope:        raw.Scope,
		TokenType:    raw.TokenType,
	}, nil
}

func newPKCEPair() (verifier, challenge string, err error) {
	verifier, err = randomString(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}
