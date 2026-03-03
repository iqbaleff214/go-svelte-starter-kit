package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// OAuthUserInfo holds the normalised user data returned by an OAuth provider.
type OAuthUserInfo struct {
	ID          string
	Email       string
	Name        string
	Picture     string
	AccessToken string
	ExpiresAt   *time.Time
}

// GoogleProvider implements the Google OAuth2 authorization code flow.
type GoogleProvider struct {
	cfg *oauth2.Config
}

func NewGoogleProvider(cfg config.GoogleConfig) *GoogleProvider {
	return &GoogleProvider{
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// AuthURL returns the Google consent-screen URL for the given state token.
func (p *GoogleProvider) AuthURL(state string) string {
	return p.cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// Exchange swaps an authorization code for user info.
func (p *GoogleProvider) Exchange(ctx context.Context, code string) (*OAuthUserInfo, error) {
	oauthToken, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange oauth code: %w", err)
	}

	client := p.cfg.Client(ctx, oauthToken)
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned %d", resp.StatusCode)
	}

	var raw struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode google userinfo: %w", err)
	}

	info := &OAuthUserInfo{
		ID:          raw.ID,
		Email:       raw.Email,
		Name:        raw.Name,
		Picture:     raw.Picture,
		AccessToken: oauthToken.AccessToken,
	}
	if !oauthToken.Expiry.IsZero() {
		t := oauthToken.Expiry
		info.ExpiresAt = &t
	}

	return info, nil
}
