package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	loginServer   = "https://magichour.ai"
	loginClient   = "magic-hour-cli"
	loginResource = "https://api.magichour.ai"
)

func randomURLToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func browserLogin(ctx context.Context, output io.Writer) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("start local login callback: %w", err)
	}
	defer listener.Close()

	verifier, err := randomURLToken()
	if err != nil {
		return "", err
	}
	state, err := randomURLToken()
	if err != nil {
		return "", err
	}
	challenge := sha256.Sum256([]byte(verifier))
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	authorizeURL, err := url.Parse(loginServer + "/oauth/authorize")
	if err != nil {
		return "", err
	}
	query := authorizeURL.Query()
	query.Set("client_id", loginClient)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	query.Set("code_challenge_method", "S256")
	query.Set("resource", loginResource)
	query.Set("state", state)
	authorizeURL.RawQuery = query.Encode()

	type result struct {
		code string
		err  error
	}
	callback := make(chan result, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(params.Get("state")), []byte(state)) != 1 || params.Get("iss") != loginServer {
			http.Error(w, "Invalid login callback", http.StatusBadRequest)
			return
		}
		response := result{code: params.Get("code")}
		if oauthError := params.Get("error"); oauthError != "" {
			response.err = fmt.Errorf("login denied: %s", oauthError)
		} else if response.code == "" {
			response.err = fmt.Errorf("login callback has no authorization code")
		}
		select {
		case callback <- response:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if response.err != nil {
				fmt.Fprintln(w, "Magic Hour login was canceled. You can close this tab.")
			} else {
				fmt.Fprintln(w, "Magic Hour login complete. You can close this tab.")
			}
		default:
			http.Error(w, "Login callback already received", http.StatusConflict)
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	defer server.Close()

	fmt.Fprintln(output, "Opening browser to log in to Magic Hour.")
	fmt.Fprintln(output, "If it does not open, visit:", authorizeURL.String())
	if err := openBrowser(ctx, authorizeURL.String()); err != nil {
		fmt.Fprintln(output, "Could not open browser:", err)
	}
	select {
	case response := <-callback:
		if response.err != nil {
			return "", response.err
		}
		return exchangeLoginCode(ctx, response.code, verifier, redirectURI)
	case err := <-serverDone:
		return "", fmt.Errorf("local login callback stopped: %w", err)
	case <-ctx.Done():
		return "", fmt.Errorf("login timed out or was canceled: %w", ctx.Err())
	}
}

func openBrowser(ctx context.Context, address string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(ctx, "open", address)
	case "windows":
		command = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", address)
	default:
		command = exec.CommandContext(ctx, "xdg-open", address)
	}
	if err := command.Start(); err != nil {
		return err
	}
	go command.Wait()
	return nil
}

func exchangeLoginCode(ctx context.Context, code, verifier, redirectURI string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {loginClient},
		"redirect_uri":  {redirectURI},
		"resource":      {loginResource},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, loginServer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("exchange login code: %w", err)
	}
	defer response.Body.Close()
	var token struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&token); err != nil {
		return "", fmt.Errorf("read login response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: %s: %s", token.Error, token.ErrorDescription)
	}
	if token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") {
		return "", fmt.Errorf("login response has no bearer token")
	}
	return token.AccessToken, nil
}
