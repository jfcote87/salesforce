// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/salesforce"
	"github.com/jfcote87/salesforce/auth/jwt"
)

func TestConfig(t *testing.T) {
	var tests = []struct {
		name string
		err  string
		cfg  *jwt.Config
	}{
		{name: "test00", err: "host may not be empty", cfg: &configs[0]},
		{name: "test01", err: "consumer_key may not be empty", cfg: &configs[1]},
		{name: "test02", err: "user_id may not be empty", cfg: &configs[2]},
		{name: "test03", err: "invalid key:", cfg: &configs[3]},
		{name: "test03", err: "invalid key:", cfg: &configs[4]},
		{name: "test04", err: "", cfg: &configs[5]},
	}

	for _, tt := range tests {
		_, err := tt.cfg.Service(nil)
		if tt.err != "" {
			if err == nil || !strings.HasPrefix(err.Error(), tt.err) {
				t.Errorf("%s: expected %s; got %v", tt.name, tt.err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: expected succes; got %v", tt.name, err)
		}
	}
}

func TestServiceFromJSON(t *testing.T) {
	_, err := jwt.ServiceFromFile("/tmp/doesnotexist.json", nil)
	if err == nil {
		t.Errorf("expecting path not found; got success")
	}
	if _, err = jwt.ServiceFromReader(errReader{}, nil); err == nil || err.Error() != "read error" {
		t.Errorf("expecting read error; got %v", err)
	}
	if _, err = jwt.ServiceFromReader(badConfig, nil); err == nil ||
		err.Error() != "unexpected end of JSON input" {
		t.Errorf("expecting unexpected end of JSON input; got %v", err)
	}
	sv, err := jwt.ServiceFromFile("testfiles/good.json", nil)
	if err != nil {
		t.Errorf("expecting success; got %v", err)
	}
	_ = sv

}

var configs = []jwt.Config{
	{},
	{
		Host: "example.my.salesforce.com",
	},
	{
		Host:        "example.my.salesforce.com",
		ConsumerKey: "abcdefghijklmnopqrstuvwxyz",
	},
	{
		Host:        "example.my.salesforce.com",
		ConsumerKey: "abcdefghijklmnopqrstuvwxyz",
		UserID:      "user@example.com",
	},
	{
		Host:        "example.my.salesforce.com",
		ConsumerKey: "abcdefghijklmnopqrstuvwxyz",
		UserID:      "user@example.com",
		Key:         testKey[0:80],
	},
	{
		Host:        "example.my.salesforce.com",
		ConsumerKey: "abcdefghijklmnopqrstuvwxyz",
		UserID:      "user@example.com",
		Key:         testKey,
	},
}

const testKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvF71ZPMJlXngPNV7PWmKWaXGjX16+5cTsoI/wIXhlS6atYh6
RMDwoaG4uuI9WeqBE6Zz7PxEstNTru1Oq7Ij0K76ia5bI+uVOjG6nYbec6ieIe69
fNfy7qN2VLF6Z/RBHm7VOiraVe8/AmrTCvMvS1pUMGwBdWXBpFtVg4BDQH9r63kZ
St2DTm6pX6cr35csFZK0YlJkomMiwCvK5Wf4vO6/lf8kOLG8l6qRm6XZ1Xo3jZyr
UQ6QKX0kSSdAFHNOT0qfEQ66e/xsRZxWY2f/diTjEB71xv1YPIHrg//y9o9BR0er
Mqp5+mOTG+S22cFTdQsG4JcwswA1L4aY5fposQIDAQABAoIBABUeBB6cDGAAeL35
JMa+tS7Vocus3IOl7SRe66y2lZJ21gsx0JsykgdcrOvufvg8jNnaGDbiFQWDIWeD
3QTshI1ZgGa88CS3vVP3zTbprriCl6+wJvf+8ZIFKzfVaaaLwF0cCbVqpm1p08N+
nEgm+Q9WggenpAY4MRwuzQhf8aoiLYVIKfUfcGowNK8gn3BkX89GoUb15XT1Rt9s
bP73EdB8p4hX23c54v7D1U/aMJq+U8wW/GunHr6EoMzbq/kgTl2aMzry7KX3LXN8
ObORh+TdVYnnwo/naiwucTcGt9zZUfYLL1WOZQZkYSizAWdiZKd6FsjCmMul6tDK
MnOerJ0CgYEA38C0Xois/LH89tYZJtgND4dZ4dfvBxJCwT3gPB/wR62OfpJ5jEzv
0chB4MmgMJoTWe/jmduYMmTtyy+CKy0OMPLmu66aHQkdF5kWIf3nL4JtmOFQtdlF
/Omvxti4C7wZsY2r3wCFXUhMUCMB6qUT9UxxEhkfL3I2PFWlBkYO8csCgYEA14Ta
As5n8bM+1FMVKQ7LKJbHPv1nFEpyjJ8ivZaZkV4JqmHvmJR+1aMX2X6WdRuFlFMG
TMazOaWj5R80FoOFm24kGCsVMODPz4LlkQ3lma+Wc9kdDQhxSFAL+IFUseWkZgnr
VWvSTVh1Jbw3bjeuTqhQbQgXZcUaKssAk1zgD/MCgYB+aX790bX55g0G35rCKVnn
pg6P29E9a4Gvb2faUCkONe3FcLefHnB3Uu51MzR/gOzh6Pfrmvb3sbHvE141SnU0
DmdxLYoAUX/QLzsj5TDR1JxavSE+PAyggN5AN3xzlMfnWiT6Dm9Kbmg+9ihFCxKl
iZRwJyVJRvuBRtm/G6Gh1QKBgQC6lVexCkVPKWGBrJQrQZV9BFxnGjc9h954A+Wt
wU4eXg18JuGpdRYBmvsw3rkflb4l1WMk4PmVNOQZntQXkbIACHDTQ6lK8ba37pkU
5bUbQrq8fQD7oY2Bj1ttv3o1sZyMgpXtFDWzpJt3GeXbU/ViP7GxU0n+X4/x8GIF
MmkBJQKBgEFKTyjjY9Xs04DtGhTIiuGnH7aKh8QMh7UW6Ca5goFwuXFAnU5tue5/
CKzIl7psc2GslwRP9X9f5zB5aNvOoXKmQBPgLJdwz0/RBwV+wp3keW3+5SX8+d49
OTUC19OpuGYQtP2zuXe4tiZvfD53sizDcRtpMfz5AB3qY9hCgvwj
-----END RSA PRIVATE KEY-----
`

var badConfig = bytes.NewReader([]byte(`{"host}`))

var errNil = errors.New("read error")

type errReader struct {
	err error
}

func (e errReader) Read(b []byte) (int, error) {
	if e.err == nil {
		e.err = errNil
	}
	return 0, e.err
}

func configFromFile(fn string) (*jwt.Config, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg *jwt.Config
	return cfg, json.NewDecoder(f).Decode(&cfg)

}

type testTransport string

func (tt testTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	oriurl := r.URL.String()
	u, err := url.Parse(string(tt))
	if err != nil {
		return nil, err
	}
	rx := *r
	rx.URL = u
	rx.Header.Set("x-origurl", oriurl)
	return http.DefaultTransport.RoundTrip(&rx)
}

type tokenCacher struct {
	tk *oauth2.Token
}

func (tc *tokenCacher) Save(ctx context.Context, tk *oauth2.Token) error {
	tc.tk = tk
	return nil
}

func (tc *tokenCacher) Get(ctx context.Context) (*oauth2.Token, error) {
	return tc.tk, nil
}

func TestConfig_Service(t *testing.T) {
	ctx := context.Background()
	srv := getTestServer(t.Errorf)
	defer srv.Close()

	cfg, err := configFromFile("testfiles/good.json")
	if err != nil {
		t.Errorf("config from file testfiles/good.json decode failed %v", err)
		return
	}
	cfg.ClientFunc = func(ctx context.Context) (*http.Client, error) {
		return &http.Client{
			Transport: testTransport(srv.URL + "/token"),
		}, nil
	}
	for _, bval := range []bool{true, false} {
		cfg.IsTest = bval
		sv, err := cfg.Service(&tokenCacher{})
		if err != nil {
			t.Errorf("service init failed: %v", err)
			continue
		}
		sv = sv.WithURL(srv.URL + "/services/")
		var recs []TestRec
		if err = sv.Query(ctx, "SELECT Id FROM TestRec", &recs); err != nil {
			t.Errorf("query expected success; got %v", err)
			continue
		}
		if len(recs) != 1 {
			t.Errorf("expected single record; got %d records", len(recs))
			continue
		}
		if ok, want := jwt.CheckTestHost(cfg.IsTest, recs[0].Auth); !ok {
			t.Errorf("%v %v expected host %s; got %s", bval, cfg.IsTest, want, recs[0].Auth)
		}
	}
}

func getTestServer(logfunc func(string, ...interface{})) *httptest.Server {
	var hf http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_ = ctx
		path := r.URL.Path

		switch path {
		case "/token":
			w.Header().Set("Content-type", "application/json")
			json.NewEncoder(w).Encode(oauth2.Token{
				AccessToken: r.Header.Get("x-origurl"),
				TokenType:   "Bearer",
			})
			return
		case "/services/query/":
			w.Header().Set("Content-type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"totalSize": 1,
				"done":      true,
				"records": []TestRec{
					{
						Attributes: &salesforce.Attributes{
							Type: "TestRec",
							URL:  "https://example.com/OBJECTID",
						},
						ID:   "OBJECTID",
						Auth: strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1),
					},
				},
			})
			return
		}
		http.Error(w, "not found", 404)
	}
	return httptest.NewServer(hf)
}

type TestRec struct {
	Attributes *salesforce.Attributes
	ID         string `json:"Id,omitempty"`
	Auth       string `json:"Auth,omitempty"`
}

func (tr TestRec) SObjectName() string {
	return "TestRec"
}

func (tr TestRec) WithAttr(ref string) salesforce.SObject {
	tr.Attributes = &salesforce.Attributes{Type: "TestRec", Ref: ref}
	return tr
}
