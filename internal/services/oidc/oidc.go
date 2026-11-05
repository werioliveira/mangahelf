package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

type Provider struct {
	config      *oauth2.Config
	userInfoURL string
}

func NewProvider(cfg ProviderConfig) *Provider {
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
		Scopes: cfg.Scopes,
	}

	return &Provider{
		config:      oauth2Config,
		userInfoURL: cfg.UserInfoURL,
	}
}

func (p *Provider) AuthURL(state string, redirectURL string) string {
	p.config.RedirectURL = redirectURL
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *Provider) Exchange(ctx context.Context, code string) (string, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (p *Provider) GetUserInfo(ctx context.Context, accessToken string) (map[string]any, error) {
	client := p.config.Client(ctx, &oauth2.Token{AccessToken: accessToken})
	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}

func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var Providers = map[string]*Provider{}

func RegisterProvider(name string, p *Provider) {
	Providers[name] = p
}

func GetProvider(name string) (*Provider, bool) {
	p, ok := Providers[name]
	return p, ok
}

func GoogleConfig(clientID, clientSecret string) *Provider {
	return NewProvider(ProviderConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
	})
}

func GitHubConfig(clientID, clientSecret string) *Provider {
	return NewProvider(ProviderConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"read:user", "user:email"},
	})
}

var stateStore = map[string]string{}
var stateStoreExpiry = map[string]time.Time{}

func StoreState(state, next string) {
	stateStore[state] = next
	stateStoreExpiry[state] = time.Now().Add(10 * time.Minute)
}

func GetAndDeleteState(state string) (string, bool) {
	next, ok := stateStore[state]
	delete(stateStore, state)
	delete(stateStoreExpiry, state)
	return next, ok
}

func CleanExpiredStates() {
	now := time.Now()
	for state, expiry := range stateStoreExpiry {
		if now.After(expiry) {
			delete(stateStore, state)
			delete(stateStoreExpiry, state)
		}
	}
}
