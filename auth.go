package main

import (
	"context"
	"encoding/json"
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
	jwks     keyfunc.Keyfunc
	cancel   context.CancelFunc
	issuer   string
	audience string
	clientID string
	scopes   string
	config   []byte
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

	jwksCtx, cancel := context.WithCancel(ctx)

	jwksURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	jwks, err := keyfunc.NewDefaultCtx(jwksCtx, []string{jwksURL})
	if err != nil {
		cancel()
		panic("failed to create JWKS keyfunc from " + jwksURL + ": " + err.Error())
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
		jwks:     jwks,
		cancel:   cancel,
		issuer:   issuer,
		audience: audience,
		clientID: clientID,
		scopes:   scopes,
		config:   configJSON,
	}
}

func (a *OIDCAuth) ValidateToken(tokenString string) error {
	_, err := jwt.Parse(tokenString, a.jwks.KeyfuncCtx(context.Background()),
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
	)
	return err
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