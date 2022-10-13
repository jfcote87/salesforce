package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jfcote87/salesforce/auth"
)

type testAuth struct {
	hasGetErr bool
	Host      string
	m         sync.Mutex
}

func (tc *testAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	var mx = make(map[string]string)
	for k, v := range r.Form {
		mx[k] = v[0]
	}
	w.Header().Set("Content-type", "application/json")
	mx["access_token"] = "NewToken"
	json.NewEncoder(w).Encode(mx)
}

func (tc *testAuth) RoundTrip(r *http.Request) (*http.Response, error) {
	r.URL.Host = tc.Host
	r.URL.Scheme = "http"
	if r.Header.Get("Authorization") > "" {
		return nil, fmt.Errorf("expected empty authorization header; got %s", r.Header.Get("Authorization"))
	}
	return http.DefaultTransport.RoundTrip(r)

}

func TestPasswordConfig_TokenSource(t *testing.T) {
	var tc = &testAuth{}
	srv := httptest.NewServer(tc)
	defer srv.Close()
	tc.Host = srv.URL[7:]
	ctx := context.Background()
	pc := &auth.PasswordConfig{
		Host:          "abc.salesforce.com",
		APIVersion:    "54",
		ClientID:      "clientid",
		ClientSecret:  "secret",
		Username:      "me",
		Password:      "badpassword",
		SecurityToken: "01283FFF",
		F: func(ctx context.Context) (*http.Client, error) {
			return &http.Client{Transport: tc}, nil
		},
	}
	ts := pc.TokenSource(nil)
	tk, err := ts.Token(ctx)
	if err != nil {
		t.Errorf("expected success on Token call; got %v", err)
		return
	}
	if tk.AccessToken != "NewToken" {
		t.Errorf("expected NewToken; got %s", tk.AccessToken)
	}
	if gtype, _ := tk.Extra("grant_type").(string); gtype != "password" {
		t.Errorf("expected grant_type of password; got %s", gtype)
	}
	if clientID, _ := tk.Extra("client_id").(string); clientID != "clientid" {
		t.Errorf("expected client_id = clientid; got %s", clientID)
	}
	if clientSecret, _ := tk.Extra("client_secret").(string); clientSecret != "secret" {
		t.Errorf("expected client_secret = secret; got %s", clientSecret)
	}
	if username, _ := tk.Extra("username").(string); username != "me" {
		t.Errorf("expected username = me; got %s", username)
	}
	if password, _ := tk.Extra("password").(string); password != "badpassword01283FFF" {
		t.Errorf("expected password = badpassword01283FFF; got %s", password)
	}

}
