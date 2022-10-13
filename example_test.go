// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce_test

import (
	"context"
	"log"

	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/oauth2/clientcredentials"
	"github.com/jfcote87/salesforce"
	"github.com/jfcote87/salesforce/auth/jwt"
)

const credentialFilename = "testfiles/example.jwt.json"

func ExampleService_Call() {
	ctx := context.Background()
	sv, err := jwt.ServiceFromFile(credentialFilename, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
	// retrieve contact records
	var records []Contact
	var qry = "SELECT Id, Name FROM Contact WHERE MailingPostalCode = '80907'"
	if err := sv.Query(ctx, qry, &records); err != nil {
		log.Fatalf("query error %v", err)
	}

	var updateRecs []salesforce.SObject

	// prepare updates
	for _, c := range records {
		c.DoNotCall = true
		updateRecs = append(updateRecs)
	}

	opResponses, err := sv.UpdateRecords(ctx, false, updateRecs)
	if err != nil {
		log.Fatalf("update error %v", err)
	}
	// loop through responses looking for errors
	for i, r := range opResponses {
		if !r.Success {
			contact, _ := updateRecs[i].(Contact)
			log.Printf("%s %s %v", contact.ContactID, contact.Name, r.Errors)
		}
	}
}

func ExampleNew_password() {
	// The example uses username/password entries to authorize the new service. More details may be found at:
	// https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_username_password_flow.htm&type=5

	cfg := &clientcredentials.Config{
		ClientID:     "<clientID>",
		ClientSecret: "<clientSecret>",
		TokenURL:     "https://login.salesforce.com/services/oauth2/token",
		EndpointParams: map[string][]string{
			"grant_type": {"<password>"},
			"username":   {"<username>"},
			"password":   {"<password>" + "<securityToken>"},
		},
	}
	sv := salesforce.New("<instance url>", "<version>", cfg.TokenSource(nil))
	var recs []Contact
	if err := sv.Query(context.Background(), "SELECT Id FROM Contact", &recs); err != nil {
		log.Fatalf("query error: %v", err)
	}
	log.Printf("total recs = %d", len(recs))

}

func ExampleNew_fromtoken() {
	tk := &oauth2.Token{
		AccessToken: "valid token from somewhere",
	}
	sv := salesforce.New("<instance url>", "<version>", oauth2.StaticTokenSource(tk))
	var recs []Contact
	if err := sv.Query(context.Background(), "SELECT Id FROM Contact", &recs); err != nil {
		log.Fatalf("query error: %v", err)
	}
	log.Printf("total recs = %d", len(recs))
}
