// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configoauth2

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	errNoClientIDProvided     = errors.New("no ClientID provided in OAuth Client Credentials configuration")
	errNoTokenURLProvided     = errors.New("no TokenURL provided in OAuth Client Credentials configuration")
	errNoClientSecretProvided = errors.New("no ClientSecret provided in OAuth Client Credentials configuration")
)

// OAuth2ClientCredentials stores the configuration for OAuth2 Client Credentials (2-legged OAuth2 flow) setup
type OAuth2ClientCredentials struct {
	// ClientID is the application's ID.
	ClientID string `mapstructure:"client_id"`

	// ClientSecret is the application's secret.
	ClientSecret string `mapstructure:"client_secret"`

	// TokenURL is the resource server's token endpoint
	// URL. This is a constant specific to each server.
	TokenURL string `mapstructure:"token_url"`

	// Scope specifies optional requested permissions.
	Scopes []string `mapstructure:"scopes"`
}

func (c *OAuth2ClientCredentials) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if c.ClientID == "" {
		return nil, errNoClientIDProvided
	}
	if c.ClientSecret == "" {
		return nil, errNoClientSecretProvided
	}
	if c.TokenURL == "" {
		return nil, errNoTokenURLProvided
	}
	config := clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		TokenURL:     c.TokenURL,
		Scopes:       c.Scopes,
	}

	return &oauth2.Transport{
		Source: config.TokenSource(context.Background()),
		Base:   base,
	}, nil
}
