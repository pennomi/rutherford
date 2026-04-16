package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Authenticator interface {
	Middleware(next http.Handler) http.Handler
	ValidateToken(token string) error
	AuthConfigJSON() []byte
	Close()
}

type OIDCAuth struct {
	issuer        string
	clientID      string
	clientSecret  string
	scopes        string
	userinfoURL   string
	allowedEmails map[string]bool
	config        []byte
}

type AuthConfig struct {
	Provider     string `json:"provider"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Scopes       string `json:"scopes"`
}

type oidcFileConfig struct {
	issuer       string
	clientID     string
	clientSecret string
	scopes       string
}

const googleIssuer = "https://accounts.google.com"

func loadAuthFile(path string) oidcFileConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("failed to read auth config file " + path + ": " + err.Error())
	}

	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	if err != nil {
		panic("failed to parse auth config file " + path + ": " + err.Error())
	}

	if wrapper, ok := raw["web"]; ok {
		return parseGoogleClient(path, "web", wrapper)
	}
	if wrapper, ok := raw["installed"]; ok {
		return parseGoogleClient(path, "installed", wrapper)
	}

	var simple struct {
		Issuer   string `json:"issuer"`
		ClientID string `json:"clientId"`
		Scopes   string `json:"scopes"`
	}
	err = json.Unmarshal(data, &simple)
	if err != nil {
		panic("failed to parse simple PKCE auth config " + path + ": " + err.Error())
	}
	if simple.Issuer == "" {
		panic("auth config " + path + " is missing required field \"issuer\"")
	}
	if simple.ClientID == "" {
		panic("auth config " + path + " is missing required field \"clientId\"")
	}
	if simple.Scopes == "" {
		simple.Scopes = "openid profile email"
	}
	return oidcFileConfig{
		issuer:   simple.Issuer,
		clientID: simple.ClientID,
		scopes:   simple.Scopes,
	}
}

func parseGoogleClient(path, key string, wrapper json.RawMessage) oidcFileConfig {
	var client struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	err := json.Unmarshal(wrapper, &client)
	if err != nil {
		panic("failed to parse Google \"" + key + "\" block in " + path + ": " + err.Error())
	}
	if client.ClientID == "" {
		panic("Google auth config " + path + " is missing \"" + key + ".client_id\"")
	}
	if client.ClientSecret == "" {
		panic("Google auth config " + path + " is missing \"" + key + ".client_secret\"")
	}
	return oidcFileConfig{
		issuer:       googleIssuer,
		clientID:     client.ClientID,
		clientSecret: client.ClientSecret,
		scopes:       "openid profile email",
	}
}

func NewOIDCAuth(authConfigPath string) *OIDCAuth {
	file := loadAuthFile(authConfigPath)

	allowedEmails := parseCommaSeparated(os.Getenv("ALLOWED_EMAILS"))

	oidcDoc, err := discoverOIDC(strings.TrimRight(file.issuer, "/") + "/.well-known/openid-configuration")
	if err != nil {
		panic("failed to discover OIDC config: " + err.Error())
	}

	cfg := AuthConfig{
		Provider:     "oidc",
		Issuer:       file.issuer,
		ClientID:     file.clientID,
		ClientSecret: file.clientSecret,
		Scopes:       file.scopes,
	}
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		panic("failed to marshal auth config: " + err.Error())
	}

	return &OIDCAuth{
		issuer:        file.issuer,
		clientID:      file.clientID,
		clientSecret:  file.clientSecret,
		scopes:        file.scopes,
		userinfoURL:   oidcDoc.UserinfoURL,
		allowedEmails: allowedEmails,
		config:        configJSON,
	}
}

func (a *OIDCAuth) ValidateToken(tokenString string) error {
	userinfo, err := a.fetchUserinfo(tokenString)
	if err != nil {
		return err
	}
	if len(a.allowedEmails) > 0 && !a.allowedEmails[userinfo.Email] {
		return fmt.Errorf("email %q is not allowed", userinfo.Email)
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

func (a *OIDCAuth) Close() {}

type NoAuth struct {
	config []byte
}

func NewNoAuth() *NoAuth {
	cfg, err := json.Marshal(AuthConfig{Provider: "none"})
	if err != nil {
		panic("failed to marshal no-auth config: " + err.Error())
	}
	return &NoAuth{config: cfg}
}

func (a *NoAuth) ValidateToken(token string) error { return nil }

func (a *NoAuth) Middleware(next http.Handler) http.Handler { return next }

func (a *NoAuth) AuthConfigJSON() []byte { return a.config }

func (a *NoAuth) Close() {}

type oidcDiscovery struct {
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
	if doc.UserinfoURL == "" {
		return oidcDiscovery{}, fmt.Errorf("userinfo_endpoint not found in openid-configuration")
	}
	return doc, nil
}

type userinfoResponse struct {
	Email string `json:"email"`
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return userinfoResponse{}, fmt.Errorf("userinfo request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var info userinfoResponse
	err = json.Unmarshal(body, &info)
	if err != nil {
		return userinfoResponse{}, err
	}
	if info.Email == "" {
		return userinfoResponse{}, fmt.Errorf("userinfo response did not include an email claim (ensure the OIDC client requests the \"email\" scope)")
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

func NewAuthenticator(noAuth bool, authConfigPath string) Authenticator {
	if noAuth {
		return NewNoAuth()
	}
	return NewOIDCAuth(authConfigPath)
}
