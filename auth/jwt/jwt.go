// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jwt provides routines for creating authenticating services using
// a jwt created using a private key.  Instructions for registering the private key
// with Salesforce is found at https://developer.salesforce.com/docs/atlas.en-us.sfdx_dev.meta/sfdx_dev/sfdx_dev_auth_jwt_flow.htm
package jwt // import github.com/jfcote87/salesforce/auth/jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/jfcote87/ctxclient"
	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/oauth2/cache"
	"github.com/jfcote87/oauth2/jws"
	"github.com/jfcote87/oauth2/jwt"
	"github.com/jfcote87/salesforce"
)

const hostTest = "https://test.salesforce.com"
const hostProduction = "https://login.salesforce.com"
const tokenPath = "/services/oauth2/token"

type testHost bool

func (h testHost) Host() string {
	if h {
		return hostTest
	}
	return hostProduction
}

// Config contains sufficient info for JWT Login
type Config struct {
	Host          string `json:"host,omitempty"`           /// salesforce host for instance
	ConsumerKey   string `json:"consumer_key,omitempty"`   // sf consumer key for application
	IsTest        bool   `json:"is_test,omitempty"`        // set to yes if using sandox
	UserID        string `json:"user_id,omitempty"`        // salesforce login for user impersonation
	Key           string `json:"key,omitempty"`            // private key pem
	KeyID         string `json:"keyid,omitempty"`          // optional
	APIVersion    string `json:"version,omitempty"`        // vXX.X, leave blank for salesforce default
	TokenDuration int    `json:"tokenDuration,omitempty"`  // in minutes
	CacheFile     string `json:"file_cache_loc,omitempty"` // path of file for use with a filecache.

	ClientFunc ctxclient.Func `json:"-"` // used for testing
}

// ServiceFromFile uses the passed file to create a Service
func ServiceFromFile(fn string, tc cache.TokenCache) (*salesforce.Service, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ServiceFromReader(f, tc)
}

// ServiceFromReader uses the passed file to create a Service
func ServiceFromReader(rdr io.Reader, tc cache.TokenCache) (*salesforce.Service, error) {
	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	return ServiceFromJSON(b, tc)
}

// ServiceFromJSON uses the passed byte array to create a Service
func ServiceFromJSON(buff []byte, tc cache.TokenCache) (*salesforce.Service, error) {
	var cx *Config
	if err := json.Unmarshal(buff, &cx); err != nil {
		return nil, err
	}
	if tc == nil && cx.CacheFile > "" {
		tc = &cache.FileCache{Filename: cx.CacheFile}
	}
	return cx.Service(tc)
}

// TokenSource returns a non-resuable tokensource based upon Config parameters
func (c *Config) TokenSource() (oauth2.TokenSource, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	key, err := jws.RS256FromPEM([]byte(c.Key), "")
	if err != nil {
		return nil, fmt.Errorf("invalid key: %v", err)
	}
	return &jwt.Config{
		Signer:         key,
		Issuer:         c.ConsumerKey,
		Audience:       testHost(c.IsTest).Host(),
		Subject:        c.UserID,
		TokenURL:       testHost(c.IsTest).Host() + tokenPath,
		HTTPClientFunc: c.ClientFunc,
	}, nil

}

var (
	errHostEmpty        = errors.New("host may not be empty")
	errConsumerKeyEmpty = errors.New("consumer_key may not be empty")
	errUserIDEmpty      = errors.New("user_id may not be empty")
)

func (c *Config) validate() error {
	if c.Host == "" {
		return errHostEmpty
	}
	if c.ConsumerKey == "" {
		return errConsumerKeyEmpty
	}
	if c.UserID == "" {
		return errUserIDEmpty
	}
	return nil
}

// Service returns api service authorizing api calls via jwt token gen
func (c *Config) Service(tc cache.TokenCache) (*salesforce.Service, error) {
	jwtts, err := c.TokenSource()
	if err != nil {
		return nil, err
	}
	if tc == nil && c.CacheFile != "" {
		tc = &cache.FileCache{Filename: c.CacheFile}
	}
	if tc == nil {
		return salesforce.New(c.Host, c.APIVersion, oauth2.ReuseTokenSource(nil, jwtts)), nil
	}
	var ops []cache.TokenSourceParam
	if c.TokenDuration > 0 {
		ops = append(ops, cache.TokenCacheDuration(time.Duration(c.TokenDuration)*time.Minute))
	}

	ccf, err := cache.New(tc, jwtts, ops...)
	if err != nil {
		return nil, err
	}
	return salesforce.New(c.Host, c.APIVersion, ccf), nil
}

// FileCache uses filesystem to cache tokens in a predetermined file
type FileCache = cache.FileCache
