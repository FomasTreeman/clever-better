package betfair

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/yourusername/clever-better/internal/config"
)

// AuthService handles Betfair authentication
type AuthService struct {
	client *BetfairClient
	logger *log.Logger
}

// LoginResponse represents the response from certificate login
type LoginResponse struct {
	SessionToken string `json:"sessionToken"`
	LoginStatus  string `json:"loginStatus"`
}

// NewAuthService creates a new auth service
func NewAuthService(client *BetfairClient, logger *log.Logger) *AuthService {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	return &AuthService{
		client: client,
		logger: logger,
	}
}

// Login performs certificate-based authentication with Betfair
func (a *AuthService) Login(ctx context.Context) error {
	cfg := a.client.GetConfig()

	a.logger.Printf("Attempting certificate-based login with username: %s", cfg.Username)

	// Perform login request
	loginResp, err := a.loginInternal(ctx, cfg)
	if err != nil {
		return NewAuthenticationError("login request failed", err)
	}

	if loginResp.LoginStatus != "SUCCESS" {
		return NewAuthenticationError(fmt.Sprintf("login failed: %s", loginResp.LoginStatus), nil)
	}

	if loginResp.SessionToken == "" {
		return NewAuthenticationError("no session token in response", nil)
	}

	// Store session token with expiry (typically 12 hours)
	expiry := time.Now().Add(12 * time.Hour)
	a.client.SetSessionToken(loginResp.SessionToken, expiry)

	a.logger.Printf("Login successful, session token obtained")
	return nil
}

// RefreshSession refreshes the session token before expiration
func (a *AuthService) RefreshSession(ctx context.Context) error {
	if !a.client.NeedsRefresh() {
		a.logger.Printf("Session token does not need refresh yet")
		return nil
	}

	a.logger.Printf("Refreshing session token")

	// Re-authenticate to get a new token
	return a.Login(ctx)
}

// Logout invalidates the current session
func (a *AuthService) Logout(ctx context.Context) error {
	a.logger.Printf("Logging out from Betfair")

	// Clear session token
	a.client.SetSessionToken("", time.Time{})

	a.logger.Printf("Logout complete")
	return nil
}

// loginInternal performs the actual certificate login request
func (a *AuthService) loginInternal(ctx context.Context, cfg *config.BetfairConfig) (*LoginResponse, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}

	// Create TLS config with client certificate
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Build login request with form data
	loginURL := "https://identitysso-cert.betfair.com/api/certlogin"

	// Create form data: username and password
	formData := url.Values{}
	formData.Set("username", cfg.Username)
	formData.Set("password", cfg.Password)

	a.logger.Printf("Sending login request to: %s", loginURL)

	// Create request with form body
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		loginURL,
		bytes.NewBufferString(formData.Encode()),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create HTTP client with TLS config
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, err
	}

	return &loginResp, nil
}

// executeRequest executes an HTTP request with TLS certificate
func executeRequest(req *http.Request, tlsConfig *tls.Config) (*http.Response, error) {
	// This would use a custom HTTP client with TLS config
	// For now, this is a placeholder
	return nil, fmt.Errorf("not implemented: executeRequest requires custom HTTP client with TLS support")
}

// Note: The above helper functions are placeholders. In production, the HTTP client
// would need to be extended to support custom TLS configurations for certificate-based auth.
// Alternatively, use a specialized library like resty with TLS cert support.
