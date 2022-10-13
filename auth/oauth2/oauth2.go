// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package oauth2 contains helper routines for 3-legged Oauth2
package oauth2 // import github.com/jfcote87/salesforce/auth/oauth2

import (
	"context"
	"net/http"

	"github.com/jfcote87/oauth2"
)

// Config provides parameters for beginning and validating oauth2 flow from
// Salesforce
type Config struct {
	*oauth2.Config
	AuthURLOptions  []oauth2.AuthCodeOption
	ExchangeOptions []oauth2.AuthCodeOption
	ValidateState   func(context.Context, string) error
	PersistState    func(context.Context) (string, error)
}

// HandleCallback verifies a callback's state and code values
func (c *Config) HandleCallback(ctx context.Context, req *http.Request) (*oauth2.Token, error) {
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	state := req.Form.Get("state")
	code := req.Form.Get("code")
	if c.ValidateState != nil {
		if err := c.ValidateState(ctx, state); err != nil {
			return nil, err
		}
	}
	return c.Config.Exchange(ctx, code, c.ExchangeOptions...)
}

// AuthURL builds the url for beginning the Salesforce oauth process.  The
// web server should redirect the user to this url
func (c *Config) AuthURL(ctx context.Context) (string, error) {
	state := ""
	if c.PersistState != nil {
		st, err := c.PersistState(ctx)
		if err != nil {
			return "", err
		}
		state = st
	}
	return c.Config.AuthCodeURL(state, c.AuthURLOptions...), nil
}
