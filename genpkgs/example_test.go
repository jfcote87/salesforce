// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package genpkgs_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/salesforce"
	"github.com/jfcote87/salesforce/genpkgs"
)

var config = &genpkgs.Config{
	StructOverrides: map[string]*genpkgs.Override{
		"Account": {
			Name: "Household",
			Fields: map[string]genpkgs.FldOverride{
				"AccountNumber": {
					Name: "HouseholdID",
				},
			},
		},
	},
	SkipRelationshipGlobal: map[string]bool{
		"CreatedById":      true,
		"LastModifiedById": true,
	},
	Packages: []genpkgs.Parameters{
		{
			Name:            "content",
			Description:     "defines all standard content sobjects",
			GoFilename:      "content/content.go",
			IncludeStandard: true,
			IncludeMatch:    "^Content",
			ReplaceMatch:    "^Content",
			ReplaceWith:     "",
		},
		{
			Name:                   "changes",
			Description:            "defines all ChangeEvent sobjects",
			GoFilename:             "changes/changes.go",
			IncludeStandard:        true,
			IncludeCustom:          true,
			AssociatedIdentityType: "ChangeEvent",
		},
		{
			Name:          "custom",
			Description:   "describes all custom sobjects",
			GoFilename:    "custom/custom.go",
			IncludeCustom: true,
		},
		{
			Name:            "standard",
			Description:     "describes all standard sobjects",
			GoFilename:      "sobjects.go",
			IncludeStandard: true,
		},
	},
}

var validOauth2Token = &oauth2.Token{
	AccessToken: "Valid Token",
}

var packagePath = "gowork/sf/sobjects"

func WritePackages() {
	ctx := context.Background()

	if err := os.Chdir(packagePath); err != nil {
		log.Fatalf("%v", err)
	}

	var instanceName = "my-instance-name"
	sv := salesforce.New(instanceName, "", oauth2.StaticTokenSource(validOauth2Token))

	// MakeSource returns a map of [file name] and a byte array of go source
	srcMap, err := config.MakeSource(ctx, sv, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
	for fn, src := range srcMap {
		if err := ioutil.WriteFile(fn, src, 0755); err != nil {
			log.Printf("error writing %s %v", fn, err)
		}
	}
}
