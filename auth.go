package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Authenticator interface {
	Middleware(next http.Handler) http.Handler
	ValidateToken(token string) error
	AuthConfigJSON() []byte
	Close()
}

type OIDCAuth struct {
	jwks          keyfunc.Keyfunc
	cancel        context.CancelFunc
	issuer        string
	audience      string
	clientID      string
	scopes        string
	userinfoURL   string
	allowedEmails map[string]bool
	allowedGroups map[string]bool
	config        []byte
}

type AuthConfig struct {
	Provider string `json:"provider"`
	Issuer   string `json:"issuer"`
	ClientID string `json:"clientId"`
	Scopes   string `json:"scopes"`
}

func NewOIDCAuth(ctx context.Context) *OIDCAuth {
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		panic("OIDC_ISSUER is required")
	}

	audience := os.Getenv("OIDC_AUDIENCE")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	if clientID == "" {
		panic("OIDC_CLIENT_ID is required")
	}
	if audience == "" {
		audience = clientID
	}

	scopes := os.Getenv("OIDC_SCOPES")
	if scopes == "" {
		scopes = "openid profile email"
	}

	allowedEmails := parseCommaSeparated(os.Getenv("OIDC_ALLOWED_EMAILS"))
	allowedGroups := parseCommaSeparated(os.Getenv("OIDC_ALLOWED_GROUPS"))

	jwksCtx, cancel := context.WithCancel(ctx)

	oidcDoc, err := discoverOIDC(strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration")
	if err != nil {
		cancel()
		panic("failed to discover OIDC config: " + err.Error())
	}
	jwks, err := keyfunc.NewDefaultCtx(jwksCtx, []string{oidcDoc.JWKSURI})
	if err != nil {
		cancel()
		panic("failed to create JWKS keyfunc from " + oidcDoc.JWKSURI + ": " + err.Error())
	}

	cfg := AuthConfig{
		Provider: "oidc",
		Issuer:   issuer,
		ClientID: clientID,
		Scopes:   scopes,
	}
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		cancel()
		panic("failed to marshal auth config: " + err.Error())
	}

	return &OIDCAuth{
		jwks:          jwks,
		cancel:        cancel,
		issuer:        issuer,
		audience:      audience,
		clientID:      clientID,
		scopes:        scopes,
		userinfoURL:   oidcDoc.UserinfoURL,
		allowedEmails: allowedEmails,
		allowedGroups: allowedGroups,
		config:        configJSON,
	}
}

func (a *OIDCAuth) ValidateToken(tokenString string) error {
	_, err := jwt.Parse(tokenString, a.jwks.KeyfuncCtx(context.Background()),
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return err
	}

	if len(a.allowedEmails) > 0 || len(a.allowedGroups) > 0 {
		userinfo, err := a.fetchUserinfo(tokenString)
		if err != nil {
			return fmt.Errorf("failed to fetch userinfo: %w", err)
		}

		if len(a.allowedEmails) > 0 {
			if !a.allowedEmails[userinfo.Email] {
				return fmt.Errorf("email %q is not allowed", userinfo.Email)
			}
		}

		if len(a.allowedGroups) > 0 {
			found := false
			for _, g := range userinfo.Groups {
				if a.allowedGroups[g] {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("user is not in any allowed group")
			}
		}
	}

	return nil
}

func (a *OIDCAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		err := a.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *OIDCAuth) AuthConfigJSON() []byte {
	return a.config
}

func (a *OIDCAuth) Close() {
	a.cancel()
}

type NoAuth struct {
	config []byte
}

func NewNoAuth() *NoAuth {
	cfg, _ := json.Marshal(AuthConfig{Provider: "none"})
	return &NoAuth{config: cfg}
}

func (a *NoAuth) ValidateToken(token string) error { return nil }

func (a *NoAuth) Middleware(next http.Handler) http.Handler { return next }

func (a *NoAuth) AuthConfigJSON() []byte { return a.config }

func (a *NoAuth) Close() {}

type oidcDiscovery struct {
	JWKSURI     string `json:"jwks_uri"`
	UserinfoURL string `json:"userinfo_endpoint"`
}

func discoverOIDC(openidConfigURL string) (oidcDiscovery, error) {
	resp, err := http.Get(openidConfigURL)
	if err != nil {
		return oidcDiscovery{}, fmt.Errorf("failed to fetch %s: %w", openidConfigURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oidcDiscovery{}, fmt.Errorf("failed to read response: %w", err)
	}
	var doc oidcDiscovery
	err = json.Unmarshal(body, &doc)
	if err != nil {
		return oidcDiscovery{}, fmt.Errorf("failed to parse openid-configuration: %w", err)
	}
	if doc.JWKSURI == "" {
		return oidcDiscovery{}, fmt.Errorf("jwks_uri not found in openid-configuration")
	}
	return doc, nil
}

type userinfoResponse struct {
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

func (a *OIDCAuth) fetchUserinfo(accessToken string) (userinfoResponse, error) {
	req, err := http.NewRequest("GET", a.userinfoURL, nil)
	if err != nil {
		return userinfoResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userinfoResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return userinfoResponse{}, err
	}
	var info userinfoResponse
	err = json.Unmarshal(body, &info)
	if err != nil {
		return userinfoResponse{}, err
	}
	return info, nil
}

func parseCommaSeparated(s string) map[string]bool {
	m := make(map[string]bool)
	if s == "" {
		return m
	}
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			m[v] = true
		}
	}
	return m
}

func claimStringSlice(claims jwt.MapClaims, key string) []string {
	raw, ok := claims[key]
	if !ok {
		return nil
	}
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func NewAuthenticator(ctx context.Context, noAuth bool) Authenticator {
	if noAuth {
		return NewNoAuth()
	}
	provider := os.Getenv("AUTH_PROVIDER")
	if provider == "" {
		provider = "oidc"
	}
	switch provider {
	case "oidc":
		return NewOIDCAuth(ctx)
	default:
		panic("unsupported AUTH_PROVIDER: " + provider + " (supported: oidc)")
	}
}