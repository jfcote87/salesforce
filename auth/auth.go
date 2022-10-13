// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package auth defines the Credentials tokensource which allows
// caching of oauth2 tokens
package auth // import github.com/jfcote87/salesforce/auth

import (
	"time"

	"github.com/jfcote87/ctxclient"
	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/salesforce"
)

const defaultTokenDuration = 4 * time.Hour
const accessTokenSandboxURL = "https://test.salesforce.com/services/oauth2/token"
const accessTokenURL = "https://login.salesforce.com/services/oauth2/token"

// PasswordConfig contains all settings needed for the username-password
// flow for special scenarios.  More details may be found at:
// https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_username_password_flow.htm&type=5
type PasswordConfig struct {
	Host          string         `json:"host,omitempty"`
	APIVersion    string         `json:"api_version,omitempty"`
	ClientID      string         `json:"client_id,omitempty"`
	ClientSecret  string         `json:"client_secret,omitempty"`
	Username      string         `json:"username,omitempty"`
	Password      string         `json:"password,omitempty"`
	SecurityToken string         `json:"security_token,omitempty"`
	ForSandbox    bool           `json:"sandbox,omitempty"`
	F             ctxclient.Func `json:"-"`
}

func tokenURL(sandbox bool) string {
	if sandbox {
		return accessTokenSandboxURL
	}
	return accessTokenURL
}

// TokenSource returns an oauth2.TokenSource using the parameters from pc
func (pc *PasswordConfig) TokenSource(tk *oauth2.Token) oauth2.TokenSource {
	oc := &oauth2.Config{
		ClientID:     pc.ClientID,
		ClientSecret: pc.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:       tokenURL(pc.ForSandbox),
			IDSecretInBody: true,
		},
		HTTPClientFunc: pc.F,
	}
	return oc.FromOptions(oauth2.SetAuthURLParam("grant_type", "password"),
		oauth2.SetAuthURLParam("username", pc.Username),
		oauth2.SetAuthURLParam("password", pc.Password+pc.SecurityToken))
}

// Service creates a service that authenticates using a token created from
// username and password.
func (pc *PasswordConfig) Service(tk *oauth2.Token) *salesforce.Service {
	ts := pc.TokenSource(tk)
	return salesforce.New(pc.Host, pc.APIVersion, oauth2.ReuseTokenSource(nil, ts))
}
