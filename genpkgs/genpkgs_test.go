// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package genpkgs_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/scanner"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/salesforce"
	"github.com/jfcote87/salesforce/genpkgs"
)

func TestOverride_StructName(t *testing.T) {

	tests := []struct {
		name string
		or   *genpkgs.Override
		want string
	}{
		// TODO: Add test cases.
		{name: "Go_Name01", or: &genpkgs.Override{Name: "ZZZZ01"}, want: "ZZZZ01"},
		{name: "Go_Name02", want: "GoName02"},
		{name: "Go_Name$&03", want: "GoName03"},
		{name: "Go_Name04", or: &genpkgs.Override{}, want: "GoName04"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want != tt.or.GoName(tt.name) {
				t.Errorf("Override.StructName(%q) = %q, want %s", tt.name, tt.or.GoName(tt.name), tt.want)
			}
		})
	}
}

func TestOverride_FieldOverride(t *testing.T) {

	tests := []struct {
		name  string
		label string
		or    *genpkgs.Override
		want  genpkgs.FldOverride
	}{
		{name: "Field_01", label: "$Label 001", or: nil, want: genpkgs.FldOverride{Name: "Label001"}},
		{name: "Field_02", label: "Label_002", or: &genpkgs.Override{}, want: genpkgs.FldOverride{Name: "Label002"}},
		{name: "Field_03", label: "Label_003", or: &genpkgs.Override{
			Fields: map[string]genpkgs.FldOverride{
				"Field_03": {Name: "ZZZZ"},
			},
		}, want: genpkgs.FldOverride{Name: "ZZZZ"}},
		{name: "Field_04", label: "Label 004?", or: &genpkgs.Override{
			Fields: map[string]genpkgs.FldOverride{
				"Field_04": {SkipRelationship: true, IsPointer: true},
			},
		}, want: genpkgs.FldOverride{Name: "Label004", SkipRelationship: true, IsPointer: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.or.FieldOverride(tt.name, tt.label)
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("Override.FieldOverride() = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestOverride_Field(t *testing.T) {

	var testOR = &genpkgs.Override{Name: "ONAME", Fields: map[string]genpkgs.FldOverride{
		"Fld_02":      {Name: "F2", IsPointer: true, SkipRelationship: true},
		"Field008__c": {Name: "F8", IsPointer: true, SkipRelationship: true},
	}}

	var fields = []salesforce.Field{
		{Name: "Field001__c", Label: "Field 01", Type: "string", Length: 80, Custom: true, Updateable: true},
		{Name: "Fld_02", Label: "F2", Type: "integer", Updateable: true},
		{Name: "Field003__c", Label: "A$ abc", Type: "string", Length: 255, Custom: true, Calculated: true},
		{Name: "Field004__c", Label: "Be A", Custom: true, Type: "Reference", Length: 18, Updateable: true, IDLookup: true},
		{Name: "Field005", Label: "Fx 5y", Calculated: true, Type: "string", Length: 40},
		{Name: "Field006__c", Label: "F#!_zyz", Updateable: true, Type: "datetime", Custom: true},
		{Name: "Field007__c", Label: "Ref Id", Custom: true, Type: "Reference", Length: 18, Updateable: true, ReferenceTo: []string{"Contact"}, RelationshipName: "RefID__r"},
		{Name: "Field008__c", Label: "Refer Id", Custom: true, Type: "Reference", Length: 18, Updateable: false, ReferenceTo: []string{"Contact"}, RelationshipName: "ReferID"},
		{Name: "Field009__c", Label: "De_fi", Type: "string", Length: 255, Updateable: true, HTMLFormatted: true},
		{Name: "Field010__c", Label: "Hotel Name", Custom: true, Type: "integer", AutoNumber: true},
		{Name: "Field011__c", Label: "External Bldg Id", Custom: true, Type: "string", Length: 20, Updateable: true, ExternalID: true},
		{Name: "Field001__c", Label: "Field 01", Type: "string", Length: 80, Custom: true, Updateable: true},
		{Name: "Fld_02", Label: "F2", Type: "integer", Updateable: true},
		{Name: "Field003__c", Label: "A$ abc", Type: "string", Length: 255, Custom: true, Calculated: true},
		{Name: "Field004__c", Label: "Be A", Custom: true, Type: "Reference", Length: 18, Updateable: true, IDLookup: true},
		{Name: "Field005", Label: "Fx 5y", Calculated: true, Type: "string", Length: 40},
		{Name: "Field006__c", Label: "F#!_zyz", Updateable: true, Type: "datetime", Custom: true},
		{Name: "Field007__c", Label: "Ref Id", Custom: true, Type: "Reference", Length: 18, Updateable: true, ReferenceTo: []string{"Contact"}, RelationshipName: "RefID__r"},
		{Name: "Field008__c", Label: "Refer Id", Custom: true, Type: "Reference", Length: 18, Updateable: false, ReferenceTo: []string{"Contact"}, RelationshipName: "ReferID"},
		{Name: "Field009__c", Label: "De_fi", Type: "string", Length: 255, Updateable: true, HTMLFormatted: true},
		{Name: "Field010__c", Label: "Hotel Name", Custom: true, Type: "integer", AutoNumber: true},
		{Name: "Field011__c", Label: "External Bldg Id", Custom: true, Type: "string", Length: 20, Updateable: true, ExternalID: true},
	}
	type args struct {
		fx               salesforce.Field
		typeNm           string
		skipRelationship bool
	}
	tests := []struct {
		or   *genpkgs.Override
		args args
		want *genpkgs.Field
	}{
		{or: nil, args: args{fx: fields[0], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field01",
				GoType:  "string",
				APIName: fields[0].Name,
				Tag:     makeTag(fields[0].Name),
				Comment: "string(80)",
			}},
		{or: testOR, args: args{fx: fields[1], typeNm: "int"},
			want: &genpkgs.Field{
				GoName:  "F2",
				GoType:  "*int",
				APIName: fields[1].Name,
				Tag:     makeTag(fields[1].Name),
				Comment: "integer",
			}},
		{or: testOR, args: args{fx: fields[2], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "AAbc",
				GoType:  "string",
				APIName: fields[2].Name,
				Tag:     makeTag(fields[2].Name),
				Comment: "[READ-ONLY CALCULATED] string(255)",
			}},
		{or: testOR, args: args{fx: fields[3], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "BeA",
				GoType:  "string",
				APIName: fields[3].Name,
				Tag:     makeTag(fields[3].Name),
				Comment: "[LOOKUP] Reference(18)",
			}},
		{or: testOR, args: args{fx: fields[4], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Fx5y",
				GoType:  "string",
				APIName: fields[4].Name,
				Tag:     makeTag(fields[4].Name),
				Comment: "[READ-ONLY CALCULATED] string(40)",
			}},
		{or: testOR, args: args{fx: fields[5], typeNm: "*salesforce.Datetime"},
			want: &genpkgs.Field{
				GoName:  "FZyz",
				GoType:  "*salesforce.Datetime",
				APIName: fields[5].Name,
				Tag:     makeTag(fields[5].Name),
				Comment: "datetime",
			}},
		{or: testOR, args: args{fx: fields[6], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "RefID",
				GoType:  "string",
				Tag:     makeTag(fields[6].Name),
				APIName: fields[6].Name,
				Comment: "Reference(18)",
				Relationship: &genpkgs.Field{
					GoName: "RefIDRel", GoType: "map[string]interface{}",
					Tag:     makeTag(fields[6].RelationshipName),
					Comment: "update with external id [Contact]",
					APIName: fields[6].RelationshipName},
			}},
		{or: testOR, args: args{fx: fields[7], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "F8",
				GoType:  "*string",
				APIName: fields[7].Name,
				Tag:     makeTag(fields[7].Name),
				Comment: "[READ-ONLY] Reference(18)",
			}},
		{or: testOR, args: args{fx: fields[8], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "DeFi",
				GoType:  "string",
				APIName: fields[8].Name,
				Tag:     makeTag(fields[8].Name),
				Comment: "[HTML] string(255)",
			}},
		{or: testOR, args: args{fx: fields[9], typeNm: "int"},
			want: &genpkgs.Field{
				GoName:  "HotelName",
				GoType:  "int",
				APIName: fields[9].Name,
				Tag:     makeTag(fields[9].Name),
				Comment: "[AUTO-NUMBER READ-ONLY] integer",
			}},
		{or: testOR, args: args{fx: fields[10], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "ExternalBldgID",
				GoType:  "string",
				APIName: fields[10].Name,
				Tag:     makeTag(fields[10].Name),
				Comment: "[ExternalID] string(20)",
			}},
		{or: nil, args: args{fx: fields[11], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field001",
				GoType:  "string",
				APIName: fields[11].Name,
				Tag:     makeTag(fields[11].Name),
				Comment: "string(80)",
			}},
		{or: testOR, args: args{fx: fields[12], typeNm: "int"},
			want: &genpkgs.Field{
				GoName:  "F2",
				GoType:  "*int",
				APIName: fields[12].Name,
				Tag:     makeTag(fields[12].Name),
				Comment: "integer",
			}},
		{or: testOR, args: args{fx: fields[13], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field003",
				GoType:  "string",
				APIName: fields[13].Name,
				Tag:     makeTag(fields[13].Name),
				Comment: "[READ-ONLY CALCULATED] string(255)",
			}},
		{or: testOR, args: args{fx: fields[14], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field004",
				GoType:  "string",
				APIName: fields[14].Name,
				Tag:     makeTag(fields[14].Name),
				Comment: "[LOOKUP] Reference(18)",
			}},
		{or: testOR, args: args{fx: fields[15], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field005",
				GoType:  "string",
				APIName: fields[15].Name,
				Tag:     makeTag(fields[15].Name),
				Comment: "[READ-ONLY CALCULATED] string(40)",
			}},
		{or: testOR, args: args{fx: fields[16], typeNm: "*salesforce.Datetime"},
			want: &genpkgs.Field{
				GoName:  "Field006",
				GoType:  "*salesforce.Datetime",
				APIName: fields[16].Name,
				Tag:     makeTag(fields[16].Name),
				Comment: "datetime",
			}},
		{or: testOR, args: args{fx: fields[17], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field007",
				GoType:  "string",
				Tag:     makeTag(fields[17].Name),
				APIName: fields[17].Name,
				Comment: "Reference(18)",
				Relationship: &genpkgs.Field{
					GoName:  "Field007Rel",
					GoType:  "map[string]interface{}",
					Tag:     makeTag(fields[17].RelationshipName),
					Comment: "update with external id [Contact]",
					APIName: fields[17].RelationshipName},
			}},
		{or: testOR, args: args{fx: fields[18], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "F8",
				GoType:  "*string",
				APIName: fields[18].Name,
				Tag:     makeTag(fields[18].Name),
				Comment: "[READ-ONLY] Reference(18)",
			}},
		{or: testOR, args: args{fx: fields[19], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field009",
				GoType:  "string",
				APIName: fields[19].Name,
				Tag:     makeTag(fields[19].Name),
				Comment: "[HTML] string(255)",
			}},
		{or: testOR, args: args{fx: fields[20], typeNm: "int"},
			want: &genpkgs.Field{
				GoName:  "Field010",
				GoType:  "int",
				APIName: fields[20].Name,
				Tag:     makeTag(fields[20].Name),
				Comment: "[AUTO-NUMBER READ-ONLY] integer",
			}},
		{or: testOR, args: args{fx: fields[21], typeNm: "string"},
			want: &genpkgs.Field{
				GoName:  "Field011",
				GoType:  "string",
				APIName: fields[21].Name,
				Tag:     makeTag(fields[21].Name),
				Comment: "[ExternalID] string(20)",
			}},
	}
	for i, tt := range tests {
		nm := fmt.Sprintf("FX%03d", i)
		goName := tt.args.fx.Label
		if i > 10 {
			goName = tt.args.fx.Name
		}
		t.Run(nm, func(t *testing.T) {
			fp := tt.or.Field(tt.args.fx, goName, tt.args.typeNm, tt.args.skipRelationship)
			if fp == nil {
				t.Errorf("%s FieldOverride is nil", nm)
				return
			}
			if !reflect.DeepEqual(*fp, *tt.want) {
				t.Errorf("%s %s Override.Field() = %#v, want %#v", tt.args.fx.Name, nm, *fp, *tt.want)
				w := *tt.want
				t.Errorf("%s %s", fp.GoName, w.GoName)
				t.Errorf("%s %s", fp.GoType, w.GoType)
				t.Errorf("%s %s", fp.APIName, w.APIName)
				t.Errorf("%s %s", fp.Tag, w.Tag)
				t.Errorf("%s %s", fp.Comment, w.Comment)
				if fp.Relationship != nil && w.Relationship != nil {
					fp := fp.Relationship
					w := w.Relationship

					t.Errorf("%s %s", fp.GoName, w.GoName)
					t.Errorf("%s %s", fp.GoType, w.GoType)
					t.Errorf("%s %s", fp.APIName, w.APIName)
					t.Errorf("%s %s", fp.Tag, w.Tag)
					t.Errorf("%s ZZZ %s", fp.Comment, w.Comment)
				}
			}
		})
	}
}

func makeTag(s string) string {
	return fmt.Sprintf("`json:\"%s,omitempty\"`", s)
}

func TestPackageParams_Validate(t *testing.T) {
	type fields struct {
		Output          string
		Description     string
		Name            string
		IncludeCustom   bool
		IncludeStandard bool
		IncludeNames    []string
		IncludeMatch    string
		ReplaceMatch    string
		ReplaceWith     string
	}
	tests := []struct {
		name   string
		fields fields
		errmsg string
	}{
		{name: "test00", fields: fields{Name: "testpkg00", IncludeMatch: "]["}, errmsg: "package testpkg00 includematch regexp compile failed"},
		{name: "test01", fields: fields{Name: "testpkg01", ReplaceMatch: "]["}, errmsg: "package testpkg01 replacematch regexp compile failed"},
		{name: "test02", fields: fields{Output: "sfpackage.go"}, errmsg: "package name not specified"},
		{name: "test03", fields: fields{Name: "testpkg03", Output: "sfpackage.go"}, errmsg: "no selection criteria specified"},
		{name: "test04", fields: fields{Name: "testpkg04", Output: "sfpackage.go", IncludeCustom: true}, errmsg: ""},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &genpkgs.Parameters{
				Description:     tt.fields.Description,
				Name:            tt.fields.Name,
				IncludeCustom:   tt.fields.IncludeCustom,
				IncludeStandard: tt.fields.IncludeStandard,
				IncludeNames:    tt.fields.IncludeNames,
				IncludeMatch:    tt.fields.IncludeMatch,
				ReplaceMatch:    tt.fields.ReplaceMatch,
				ReplaceWith:     tt.fields.ReplaceWith,
			}
			var errmsg = ""
			if _, _, _, err := p.Validate(); err != nil {
				errmsg = err.Error()
			}
			if !strings.HasPrefix(errmsg, tt.errmsg) {
				t.Errorf("PackageParams.Validate() error = %v, wantErr %v", errmsg, tt.errmsg)
			}
		})
	}
}

func TestJob_TemplateData(t *testing.T) {
	pkgs := []*genpkgs.Parameters{
		{Name: "p0000", Description: "pkg description"},
		{Name: "p0001", Description: "pkg description"},
	}
	strx := [][]genpkgs.Struct{
		{
			{GoName: "ZZZ"},
			{GoName: "XXX"},
			{GoName: "AAA"},
			{GoName: "YYY"},
		},
		{
			{GoName: "ZZZ"},
			{GoName: "AAA"},
			{GoName: "AAA"},
			{GoName: "YYY"},
		},
	}
	want := []*genpkgs.TemplateData{
		{Name: "p0000", Description: "pkg description", Structs: []genpkgs.Struct{
			{GoName: "AAA"},
			{GoName: "XXX"},
			{GoName: "YYY"},
			{GoName: "ZZZ"},
		},
		},
		{Name: "p0001", Description: "pkg description", Structs: []genpkgs.Struct{
			{GoName: "AAA"},
			{GoName: "AAA_001"},
			{GoName: "YYY"},
			{GoName: "ZZZ"},
		},
		},
	}

	for idx, p := range pkgs {
		job := &genpkgs.Job{
			StructMap: map[*genpkgs.Parameters][]genpkgs.Struct{
				pkgs[idx]: strx[idx],
			},
			Config: &genpkgs.Config{},
		}
		got := job.TemplateData(p)
		if got == nil || want[idx] == nil {
			if got != nil || want[idx] != nil {
				t.Errorf("Job.TemplateData() = %v; want %v", got, want[idx])
			}
			continue
		}
		got.Duplicates = ""
		if !reflect.DeepEqual(*got, *want[idx]) {
			t.Errorf("Job.TemplateData() = %#v, want %#v", *got, *want[idx])
		}
	}
}

func TestJob_Match(t *testing.T) {
	/*type fields struct {
		IncludeCustom          bool
		IncludeStandard        bool
		AssociatedIdentityType string
		IncludeNames           []string
		IncludeMatch           string
		ReplaceMatch           string
		ReplaceWith            string
	}*/
	chgEvent := "ChangeEvent"

	pkgs := []*genpkgs.Parameters{
		{Name: "PKG00", IncludeCustom: true, IncludeStandard: true},
		{Name: "PKG01", IncludeCustom: true},
		{Name: "PKG02", IncludeStandard: true},
		{Name: "PKG03", IncludeStandard: true, AssociatedIdentityType: chgEvent},
		{Name: "PKG04", IncludeMatch: "Account", IncludeNames: []string{"Contact", "Account"}},
		{Name: "PKG05", IncludeStandard: true, AssociatedIdentityType: "ChangeEvent", IncludeMatch: "^Contact"},
	}
	job := &genpkgs.Job{
		Include:     make(map[*genpkgs.Parameters]*regexp.Regexp),
		Replace:     make(map[*genpkgs.Parameters]*regexp.Regexp),
		ReplaceText: make(map[*genpkgs.Parameters]string),
	}
	for _, p := range pkgs {
		include, replace, text, err := p.Validate()
		if err != nil {
			t.Errorf("%s expected validation %v", p.Name, err)
		}
		if include != nil {
			job.Include[p] = include
		}
		if replace != nil {
			job.Replace[p] = replace
			job.ReplaceText[p] = text
		}
	}
	type args struct {
		pkg *genpkgs.Parameters
		obj *salesforce.SObjectDefinition
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test00", args: args{pkg: pkgs[0], obj: &salesforce.SObjectDefinition{Name: "Contact"}}, want: true},
		{name: "test01", args: args{pkg: pkgs[1], obj: &salesforce.SObjectDefinition{Name: "Account"}}, want: false},
		{name: "test02", args: args{pkg: pkgs[1], obj: &salesforce.SObjectDefinition{Name: "Accounts__c", Custom: true}}, want: true},
		{name: "test03", args: args{pkg: pkgs[2], obj: &salesforce.SObjectDefinition{Name: "Account"}}, want: true},
		{name: "test04", args: args{pkg: pkgs[2], obj: &salesforce.SObjectDefinition{Name: "Accounts__c", Custom: true}}, want: false},
		{name: "test05", args: args{pkg: pkgs[3], obj: &salesforce.SObjectDefinition{Name: "AccountChangeEvent", AssociateEntityType: &chgEvent, Custom: false}}, want: true},
		{name: "test06", args: args{pkg: pkgs[3], obj: &salesforce.SObjectDefinition{Name: "Affiliation__ChangeEvent", AssociateEntityType: &chgEvent, Custom: true}}, want: false},
		{name: "test07", args: args{pkg: pkgs[4], obj: &salesforce.SObjectDefinition{Name: "Account"}}, want: true},
		{name: "test08", args: args{pkg: pkgs[4], obj: &salesforce.SObjectDefinition{Name: "Contact"}}, want: true},
		{name: "test09", args: args{pkg: pkgs[5], obj: &salesforce.SObjectDefinition{Name: "AccountChangeEvent", AssociateEntityType: &chgEvent, Custom: false}}, want: false},
		{name: "test10", args: args{pkg: pkgs[5], obj: &salesforce.SObjectDefinition{Name: "ContactChangeEvent", AssociateEntityType: &chgEvent, Custom: true}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := job.Match(tt.args.pkg, tt.args.obj); got != tt.want {
				t.Errorf("%s: Job.Match() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

var (
	// AssociateEntityChangeEvent
	aetChangeEvent = "ChangeEvent"
)

var testObjMap = map[string]salesforce.SObjectDefinition{
	"Account": {Name: "Account", Label: "Account", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Account Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "Name", Label: "Name", SoapType: "xsd:string", Type: "string", Length: 128, Updateable: true},
		{Name: "Type", Label: "Account Type", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"Contact": {Name: "Contact", Label: "People", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Contact Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "AccountId", Label: "Account Id", SoapType: "tns:ID", Type: "reference", Length: 18,
			RelationshipName: "Account", ReferenceTo: []string{"Account"}, Updateable: true},
		{Name: "FirstName", Label: "First Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
		{Name: "First_Name__c", Label: "First Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"Cust__c": {Name: "Cust__c", Label: "New Customer", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Record Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "ContactId", Label: "Contact Id", SoapType: "tns:ID", Type: "reference", Length: 18,
			RelationshipName: "Contact", ReferenceTo: []string{"Contact"}, Updateable: true},
		{Name: "Name", Label: "Cust Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"ContactChangeEvent": {Name: "ContactChangeEvent", Queryable: true, Label: "Contact Change Event", AssociateEntityType: &aetChangeEvent, AssociateParentEntity: "Contact", Fields: []salesforce.Field{
		{Name: "Id", Label: "Contact Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "AccountId", Label: "Account Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "FirstName", Label: "First Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"AccountChangeEvent": {Name: "AccountChangeEvent", Queryable: true, Label: "Account Change Event", AssociateEntityType: &aetChangeEvent, AssociateParentEntity: "Account", Fields: []salesforce.Field{
		{Name: "Id", Label: "Account Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "Name", Label: "Name", SoapType: "xsd:string", Type: "string", Length: 128, Updateable: true},
		{Name: "Type", Label: "Account Type", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"Industry__c": {Name: "Industry__c", Custom: true, Label: "Industry", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Record Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "AccountId", Label: "Account Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true,
			RelationshipName: "Account", ReferenceTo: []string{"Account"}},
		{Name: "Name", Label: "Cust Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"ContentVersion": {Name: "ContentVersion", Label: "Content Version", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Record Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "Type", Label: "Document Type", SoapType: "tns:ID", Type: "picklist", Length: 12, Updateable: true},
		{Name: "Name", Label: "Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"ContentDocument": {Name: "ContentDocument", Label: "Content Document", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Record Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "Type", Label: "Document Type", SoapType: "tns:ID", Type: "picklist", Length: 12,
			RelationshipName: "DocumentType", ReferenceTo: []string{"DocumentType"}},
		{Name: "Name", Label: "Name", SoapType: "xsd:string", Type: "string", Length: 80, Updateable: true},
	}},
	"Skip__c": {Name: "Skip__c", Custom: true, Label: "Skip Object", Updateable: true, Fields: []salesforce.Field{
		{Name: "Id", Label: "Record Id", SoapType: "tns:ID", Type: "reference", Length: 18, Updateable: true},
		{Name: "Type", Label: "Document Type", SoapType: "tns:ID", Type: "picklist", Length: 12, Updateable: true},
	}},
}

func TestJob_Struct(t *testing.T) {

	objnames := []string{"Account", "Contact", "Cust__c", "ContactChangeEvent", "AccountChangeEvent", "Industry__c"}

	pckdefs := []genpkgs.Parameters{
		{UseLabel: true},
		{UseLabel: true},
		{UseLabel: true},
		{UseLabel: true},
		{AssociatedIdentityType: "ChangeEvent", IncludeStandard: true, UseLabel: false},
		{UseLabel: false},
	}

	cfg := &genpkgs.Config{
		//FieldNamesFromLabel: true,
		StructOverrides: map[string]*genpkgs.Override{
			"Cust__c": {
				Name: "Customer",
				Fields: map[string]genpkgs.FldOverride{
					"ContactId": {
						Name: "Contact",
					},
				},
			},
			"Account": {
				Name: "VendorAccount",
				Fields: map[string]genpkgs.FldOverride{
					"Name": {
						Name: "VendorName",
					},
					"Type": {
						Name: "VendorType",
					},
				},
			},
		},
	}
	want := []genpkgs.Struct{
		{
			GoName: "VendorAccount", Label: "Account", APIName: "Account", Receiver: "v", FieldProps: []*genpkgs.Field{
				{GoName: "AccountID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "VendorName", GoType: "string", APIName: "Name", Tag: "`json:\"Name,omitempty\"`", Comment: "string(128)"},
				{GoName: "VendorType", GoType: "string", APIName: "Type", Tag: "`json:\"Type,omitempty\"`", Comment: "string(80)"},
			},
		},
		{
			GoName: "People", Label: "People", APIName: "Contact", Receiver: "p", FieldProps: []*genpkgs.Field{
				{GoName: "ContactID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "AccountID", GoType: "string", APIName: "AccountId", Tag: "`json:\"AccountId,omitempty\"`", Comment: "reference(18)"},
				{GoName: "FirstName", GoType: "string", APIName: "FirstName", Tag: "`json:\"FirstName,omitempty\"`", Comment: "string(80)"},
				{GoName: "FirstName_DUP000", GoType: "string", APIName: "First_Name__c", Tag: "`json:\"First_Name__c,omitempty\"`", Comment: "string(80)"},
			},
		},
		{
			GoName: "Customer", Label: "New Customer", APIName: "Cust__c", Receiver: "c", FieldProps: []*genpkgs.Field{
				{GoName: "RecordID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "Contact", GoType: "string", APIName: "ContactId", Tag: "`json:\"ContactId,omitempty\"`", Comment: "reference(18)"},
				{GoName: "CustName", GoType: "string", APIName: "Name", Tag: "`json:\"Name,omitempty\"`", Comment: "string(80)"},
			},
		},
		{
			GoName: "ContactChangeEvent", Label: "Contact Change Event", APIName: "ContactChangeEvent", Receiver: "c", FieldProps: []*genpkgs.Field{
				{GoName: "ContactID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "AccountID", GoType: "string", APIName: "AccountId", Tag: "`json:\"AccountId,omitempty\"`", Comment: "reference(18)"},
				{GoName: "FirstName", GoType: "string", APIName: "FirstName", Tag: "`json:\"FirstName,omitempty\"`", Comment: "string(80)"},
			},
		},
		{
			GoName: "VendorAccount", Label: "Account Change Event", APIName: "AccountChangeEvent", Receiver: "v", FieldProps: []*genpkgs.Field{
				{GoName: "ID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "VendorName", GoType: "string", APIName: "Name", Tag: "`json:\"Name,omitempty\"`", Comment: "string(128)"},
				{GoName: "VendorType", GoType: "string", APIName: "Type", Tag: "`json:\"Type,omitempty\"`", Comment: "string(80)"},
			},
		},
		{
			GoName: "Industry", Label: "Industry", APIName: "Industry__c", Receiver: "i", FieldProps: []*genpkgs.Field{
				{GoName: "ID", GoType: "string", APIName: "Id", Tag: "`json:\"Id,omitempty\"`", Comment: "reference(18)"},
				{GoName: "AccountID", GoType: "string", APIName: "AccountId", Tag: "`json:\"AccountId,omitempty\"`", Comment: "reference(18)",
					Relationship: &genpkgs.Field{GoName: "AccountIDRel", GoType: "map[string]interface{}", APIName: "Account", Tag: "`json:\"Account,omitempty\"`", Comment: "update with external id [Account]"}}, //JFC
				{GoName: "Name", GoType: "string", APIName: "Name", Tag: "`json:\"Name,omitempty\"`", Comment: "string(80)"},
			},
		},
	}

	job := &genpkgs.Job{
		Config:  cfg,
		ObjMap:  testObjMap,
		TypeMap: typeMap,
	}

	for i, nm := range objnames {
		objdef := testObjMap[nm]
		p := &pckdefs[i]

		testNm := fmt.Sprintf("test%02d", i)
		t.Run(testNm, func(t *testing.T) {
			structs := job.Struct(p, &objdef)

			wx := want[i]
			if wx.GoName != structs.GoName {
				t.Errorf("%s: wanted Name %s; got %s", nm, wx.GoName, structs.GoName)
			}
			if wx.Label != structs.Label {
				t.Errorf("%s: wanted Label %s; got %s", nm, wx.Label, structs.Label)
			}
			if wx.APIName != structs.APIName {
				t.Errorf("%s: wanted APIName %s; got %s", nm, wx.APIName, structs.APIName)
			}
			if wx.Receiver != structs.Receiver {
				t.Errorf("%s: wanted Receiver %s; got %s", nm, wx.Receiver, structs.Receiver)
			}
			if len(wx.FieldProps) != len(structs.FieldProps) {
				t.Errorf("%s: wanted %d Field Properties; got %d", nm, len(wx.FieldProps), len(structs.FieldProps))
				return
			}
			for idx := range wx.FieldProps {
				for _, msg := range checkFieldProps(nm, *wx.FieldProps[idx], *structs.FieldProps[idx]) {
					t.Error(msg)
				}
			}
		})
	}
}

func checkFieldProps(nm string, wantProp, haveProp genpkgs.Field) []string {
	var msgs []string
	if wantProp.GoName != haveProp.GoName {
		msgs = append(msgs, fmt.Sprintf("%s %s: wanted field Name = %s; got %s", wantProp.GoName, nm, wantProp.GoName, haveProp.GoName))
	}
	if wantProp.GoType != haveProp.GoType {
		msgs = append(msgs, fmt.Sprintf("%s %s: wanted field Type = %s; got %s", wantProp.GoName, nm, wantProp.GoType, haveProp.GoType))
	}
	if wantProp.Tag != haveProp.Tag {
		msgs = append(msgs, fmt.Sprintf("%s %s: wanted field Tag = %s; got %s", wantProp.GoName, nm, wantProp.Tag, haveProp.Tag))
	}
	if wantProp.APIName != haveProp.APIName {
		msgs = append(msgs, fmt.Sprintf("%s %s: wanted field APIName = %s; got %s", wantProp.GoName, nm, wantProp.APIName, haveProp.APIName))
	}
	if wantProp.Comment != haveProp.Comment {
		msgs = append(msgs, fmt.Sprintf("%s %s: wanted field Type = %s; got %s", wantProp.GoName, nm, wantProp.Comment, haveProp.Comment))
	}
	if wantProp.Relationship != nil && haveProp.Relationship == nil {
		msgs = append(msgs, fmt.Sprintf("expected relationship = %v; got %v", wantProp.Relationship, haveProp.Relationship))
	}
	return msgs
}

var typeMap = map[string]string{
	"StringList":              "string",
	"tns:ID":                  "string",
	"urn:JunctionIdListNames": "[]string]",
	"xsd:base64Binary":        "*salesforce.Binary",
	"xsd:boolean":             "bool",
	"xsd:date":                "*salesforce.Date",
	"xsd:dateTime":            "*salesforce.Datetime",
	"xsd:double":              "float64",
	"xsd:int":                 "int",
	"xsd:long":                "int",
	"xsd:string":              "string",
	"xsd:time":                "*salesforce.Time",
}

var (
	AssociateEntityChangeEvent = "ChangeEvent"
	AssociateEntityFeed        = "Feed"
	AssociateEntityShare       = "Share"
	AssociateEntityHistory     = "History"
)

var testConfig = &genpkgs.Config{
	//FieldNamesFromLabel: true,
	SkipRelationshipGlobal: map[string]bool{
		"CreatedById":      true,
		"LastModifiedById": true,
	},
	Packages: []genpkgs.Parameters{
		{
			Description:            "contains definitions for all ChangeEvent SObjects",
			Name:                   "changelog",
			IncludeCustom:          true,
			IncludeStandard:        true,
			AssociatedIdentityType: "ChangeEvent",
			UseLabel:               true,
		},
		{
			Description:     "contains definitions for an content objects (new file and note structures)",
			Name:            "content",
			IncludeStandard: true,
			IncludeMatch:    "^Content",
			ReplaceMatch:    "^Content",
			ReplaceWith:     "",
			UseLabel:        true,
		},
		{
			Description:   "contains custom object definitions",
			Name:          "custom",
			IncludeCustom: true,
			UseLabel:      true,
		},
		{
			Description:     "contains standard object definitions",
			Name:            "sobjects",
			IncludeStandard: true,
			UseLabel:        true,
		},
	},
}

func getConfigMakeTemplateDataServer(ch chan struct{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ch:

		default:
			http.Error(w, "503 err", http.StatusInternalServerError)
			return
		}
		parts := strings.Split(r.URL.Path, "/")
		if strings.HasSuffix(r.URL.Path, "/describe") {
			objnm := parts[len(parts)-2]
			obj, ok := testObjMap[objnm]
			if !ok {
				http.Error(w, fmt.Sprintf("invalid object name %s", objnm), 400)
				return
			}
			b, _ := json.MarshalIndent(obj, "", "    ")
			w.Write(b)
			return
		}
		var objs = make([]salesforce.SObjectDefinition, 0, len(testObjMap))
		for _, v := range testObjMap {
			r := v
			objs = append(objs, r)
		}
		var result = struct {
			Encoding     string                         `json:"encoding,omitempty"`
			MaxBatchSize int                            `json:"maxBatchSize,omitempty"`
			Objects      []salesforce.SObjectDefinition `json:"sobjects,omitempty"`
		}{
			Encoding:     "application/json",
			MaxBatchSize: 200,
			Objects:      objs,
		}
		json.NewEncoder(w).Encode(result)
	}))
}

func TestConfig_MakeTemplateData(t *testing.T) {
	ch := make(chan struct{})

	myCfg := *testConfig
	cfg := &myCfg
	cfg.SkipObjects = []string{"Skip__c"}
	cfg.StructOverrides = map[string]*genpkgs.Override{
		"Cust__c": {
			Name: "Customer",
			Fields: map[string]genpkgs.FldOverride{
				"ContactId": {
					Name: "Contact",
				},
			},
		},
		"Account": {
			Name: "VendorAccount",
			Fields: map[string]genpkgs.FldOverride{
				"Name": {
					Name: "VendorName",
				},
				"Type": {
					Name: "VendorType",
				},
			},
		},
	}
	srv := getConfigMakeTemplateDataServer(ch)
	defer srv.Close()
	ctx := context.Background()
	sv := salesforce.New("", "", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "ABC"})).
		WithURL(srv.URL + "/services/data/53/")

	expectedErr := "package[1] (name=) validation failed package name not specified"
	svpkg := cfg.Packages[1].Name
	cfg.Packages[1].Name = ""
	if _, err := cfg.MakeTemplateData(ctx, sv); err == nil || err.Error() != expectedErr {
		t.Errorf("expected error %s; got %v", expectedErr, err)
		return
	}
	cfg.Packages[1].Name = svpkg

	expectedErr = "object list failed"
	if _, err := cfg.MakeTemplateData(ctx, sv); err == nil || !strings.HasPrefix(err.Error(), expectedErr) {
		t.Errorf("expected error %s; got %v", expectedErr, err)
		return
	}

	close(ch) // stop errors on web

	tds, err := cfg.MakeTemplateData(ctx, sv)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	for idx, td := range tds {
		want := *wantMakeTemplateData[idx]
		got := *td
		if !reflect.DeepEqual(got, want) {
			for idx := range got.Structs {
				doDeepTest(td.Name, want.Structs[idx], got.Structs[idx], t.Errorf)
			}
		}
	}
}

func doDeepTest(nm string, ws, gs genpkgs.Struct, f func(string, ...interface{})) {
	for ix, gf := range gs.FieldProps {
		wf := ws.FieldProps[ix]
		if !reflect.DeepEqual(gf, wf) {
			f("deep %s %s %s %s", nm, gs.GoName, gf.GoName, wf.GoName)
			f("%#v", wf.Relationship)
			f("%#v", gf.Relationship)
		}
		if gf.GoName != wf.GoName || gf.APIName != wf.APIName || gf.Comment != wf.Comment {
			f("%s: %s: %s: %s %s %s %s", nm, gs.GoName, ws.GoName, gf.GoName, wf.GoName, gf.GoType, wf.GoType)
		}
	}
}

func getTestServer(t *testing.T) (*httptest.Server, []salesforce.SObjectDefinition) {
	var objs = make([]salesforce.SObjectDefinition, 0, len(testObjMap))
	for _, v := range testObjMap {
		r := v
		objs = append(objs, r)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if strings.HasSuffix(r.URL.Path, "/describe") {
			objnm := parts[len(parts)-2]
			obj, ok := testObjMap[objnm]
			if !ok {
				t.Fatalf("invalid object name %s", objnm)
			}
			b, _ := json.MarshalIndent(obj, "", "    ")
			w.Write(b)
			return
		}
		var objs = make([]salesforce.SObjectDefinition, 0, len(testObjMap))
		for _, v := range testObjMap {
			r := v
			objs = append(objs, r)
		}
		var result = struct {
			Encoding     string                         `json:"encoding,omitempty"`
			MaxBatchSize int                            `json:"maxBatchSize,omitempty"`
			Objects      []salesforce.SObjectDefinition `json:"sobjects,omitempty"`
		}{
			Encoding:     "application/json",
			MaxBatchSize: 200,
			Objects:      objs,
		}
		json.NewEncoder(w).Encode(result)
	}))
	return srv, objs

}

func TestJob_AssignSObjects(t *testing.T) {
	cfg := &genpkgs.Config{}

	srv, objs := getTestServer(t)
	ctx := context.Background()
	sv := salesforce.New("", "", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "ABC"})).
		WithURL(srv.URL + "/services/data/53/")

	defer srv.Close()

	job, err := cfg.CreateJob(ctx, sv)
	if err != nil {
		t.Errorf("createjob failed %v", err)
		return
	}
	for _, obj := range objs {
		t.Run(obj.Name, func(t *testing.T) {
			if err := job.AssignSObjects(ctx, sv, obj); err != nil {
				t.Errorf("expected success with %s; got %v", obj.Name, err)
			}
		})
	}

}

var wantMakeTemplateData = []*genpkgs.TemplateData{
	{
		Name:        "changelog",
		Description: "contains definitions for all ChangeEvent SObjects",
		Structs: []genpkgs.Struct{
			{
				GoName:   "ContactChangeEvent",
				Label:    "Contact Change Event",
				APIName:  "ContactChangeEvent",
				Receiver: "c",
				Readonly: true,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "ContactID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "AccountID",
						APIName: "AccountId",
						GoType:  "string",
						Tag:     "`json:\"AccountId,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "FirstName",
						APIName: "FirstName",
						GoType:  "string",
						Tag:     "`json:\"FirstName,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
			{
				GoName:   "VendorAccount",
				Label:    "Account Change Event",
				APIName:  "AccountChangeEvent",
				Receiver: "v",
				Readonly: true,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "AccountID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "VendorName",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(128)",
					},
					{
						GoName:  "VendorType",
						APIName: "Type",
						GoType:  "string",
						Tag:     "`json:\"Type,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
		},
	},
	{
		Name:        "content",
		Description: "contains definitions for an content objects (new file and note structures)",
		Structs: []genpkgs.Struct{
			{
				GoName:   "Document",
				Label:    "Content Document",
				APIName:  "ContentDocument",
				Receiver: "d",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "RecordID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "DocumentType",
						APIName: "Type",
						GoType:  "string",
						Tag:     "`json:\"Type,omitempty\"`",
						Comment: "[READ-ONLY] picklist(12)",
					},
					{
						GoName:  "Name",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
			{
				GoName:   "Version",
				Label:    "Content Version",
				APIName:  "ContentVersion",
				Receiver: "v",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "RecordID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "DocumentType",
						APIName: "Type",
						GoType:  "string",
						Tag:     "`json:\"Type,omitempty\"`",
						Comment: "picklist(12)",
					},
					{
						GoName:  "Name",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
		},
	}, {
		Name:        "custom",
		Description: "contains custom object definitions",
		Structs: []genpkgs.Struct{
			{
				GoName:   "Industry",
				Label:    "Industry",
				APIName:  "Industry__c",
				Receiver: "i",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "RecordID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "AccountID",
						APIName: "AccountId",
						GoType:  "string",
						Tag:     "`json:\"AccountId,omitempty\"`",
						Comment: "reference(18)",
						Relationship: &genpkgs.Field{
							GoName:  "AccountIDRel",
							APIName: "Account",
							GoType:  "map[string]interface{}",
							Tag:     "`json:\"Account,omitempty\"`",
							Comment: "update with external id [Account]",
						},
					},
					{
						GoName:  "CustName",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
		},
	},
	{
		Name:        "sobjects",
		Description: "contains standard object definitions",
		Structs: []genpkgs.Struct{
			{
				GoName:   "Customer",
				Label:    "New Customer",
				APIName:  "Cust__c",
				Receiver: "c",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "RecordID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "Contact",
						APIName: "ContactId",
						GoType:  "string",
						Tag:     "`json:\"ContactId,omitempty\"`",
						Comment: "reference(18)",
						Relationship: &genpkgs.Field{
							GoName:  "ContactRel",
							APIName: "Contact",
							GoType:  "map[string]interface{}",
							Tag:     "`json:\"Contact,omitempty\"`",
							Comment: "update with external id [Contact]", //JFC
						},
					},
					{
						GoName:  "CustName",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
			{
				GoName:   "People",
				Label:    "People",
				APIName:  "Contact",
				Receiver: "p",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "ContactID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "AccountID",
						APIName: "AccountId",
						GoType:  "string",
						Tag:     "`json:\"AccountId,omitempty\"`",
						Comment: "reference(18)",
						Relationship: &genpkgs.Field{
							GoName:  "AccountIDRel",
							APIName: "Account",
							GoType:  "map[string]interface{}",
							Tag:     "`json:\"Account,omitempty\"`",
							Comment: "update with external id [Account]",
						},
					},
					{
						GoName:  "FirstName",
						APIName: "FirstName",
						GoType:  "string",
						Tag:     "`json:\"FirstName,omitempty\"`",
						Comment: "string(80)",
					},
					{
						GoName:  "FirstName_DUP000",
						APIName: "First_Name__c",
						GoType:  "string",
						Tag:     "`json:\"First_Name__c,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
			{
				GoName:   "VendorAccount",
				Label:    "Account",
				APIName:  "Account",
				Receiver: "v",
				Readonly: false,
				FieldProps: []*genpkgs.Field{
					{
						GoName:  "AccountID",
						APIName: "Id",
						GoType:  "string",
						Tag:     "`json:\"Id,omitempty\"`",
						Comment: "reference(18)",
					},
					{
						GoName:  "VendorName",
						APIName: "Name",
						GoType:  "string",
						Tag:     "`json:\"Name,omitempty\"`",
						Comment: "string(128)",
					},
					{
						GoName:  "VendorType",
						APIName: "Type",
						GoType:  "string",
						Tag:     "`json:\"Type,omitempty\"`",
						Comment: "string(80)",
					},
				},
			},
		},
	},
}

func TestErrorList_Error(t *testing.T) {
	tests := []struct {
		name string
		el   genpkgs.ErrorList
		want string
	}{
		{name: "test00", el: genpkgs.ErrorList{errors.New("A"), errors.New("B")}, want: "A\nB"},
		{name: "test01", el: genpkgs.ErrorList{}, want: ""},
		{name: "test02", el: nil, want: ""},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.el.Error(); got != tt.want {
				t.Errorf("ErrorList.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_MakeSource(t *testing.T) {
	cfg := genpkgs.Config{
		Packages: []genpkgs.Parameters{
			{
				Description:     "Standard",
				Name:            "sobjects",
				GoFilename:      "sobjects.go",
				IncludeStandard: true,
			},
			{
				Description:   "Custom",
				Name:          "custom",
				GoFilename:    "custom/custom.go",
				IncludeCustom: true,
			},
			{
				Description:   "blank",
				Name:          "blank",
				GoFilename:    "blank.go",
				IncludeCustom: true,
				IncludeNames:  []string{"Contact"},
			},
		},
	}
	srv, _ := getTestServer(t)

	ctx := context.Background()
	sv := salesforce.New("", "", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "ABC"})).
		WithURL(srv.URL + "/services/data/53/")

	mx, err := cfg.MakeSource(ctx, sv, nil)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	var tfiles []string
	for k := range mx {
		tfiles = append(tfiles, k)
	}

	if len(mx) != 2 || mx["sobjects.go"] == nil || mx["custom/custom.go"] == nil {
		t.Errorf("expected files named sobjects.go and custom/custom.go; got %v", tfiles)
	}

	badTmpl, _ := template.New("bad").Parse("{{ .Q }}")
	_, err = cfg.MakeSource(ctx, sv, badTmpl)
	if !errors.As(err, &template.ExecError{}) {
		t.Errorf("expected template.ExecError; got %v", err)
	}

	srcTmpl, _ := template.New("src").Parse(`package a/a/a/a/
	
	func a() {}`)
	_, err = cfg.MakeSource(ctx, sv, srcTmpl)
	if !errors.As(err, &scanner.ErrorList{}) {
		t.Errorf("expected scanner.ErrorList; got %v", err)
	}

}
