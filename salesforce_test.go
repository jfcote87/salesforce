// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jfcote87/salesforce"
)

func TestRecordSlice(t *testing.T) {
	var rs = &salesforce.RecordSlice{}
	buff := []byte("xz[]")
	if err := rs.UnmarshalJSON(buff); err == nil || err.Error() != "uninitialized QueryResult" {
		t.Errorf("expected uninitialized QueryResult; got %v", err)
		return
	}

	var rx []Contact
	rs, _ = salesforce.NewRecordSlice(&rx)
	if err := rs.UnmarshalJSON(buff); err == nil || !strings.HasPrefix(err.Error(), "invalid character") {
		t.Errorf("expected invalid character;; got %v", err)
		return
	}

	if b, err := rs.MarshalJSON(); b != nil || err != nil {
		t.Errorf("expected nil value and nil error; got %s and %v", b, err)
	}

}

func TestDateTime_MarshalText(t *testing.T) {
	var d salesforce.Datetime = "2021-12-12T01:01:01.000Z"
	var dx *salesforce.Datetime
	if err := dx.UnmarshalText([]byte("\"ABCDE\"")); err == nil || err.Error() != "nil pointer" {
		t.Errorf("expected nil pointer; got %v", err)
		return
	}
	dx = &d

	if err := dx.UnmarshalText([]byte("2021-12-12T01:01:01.000Z")); err != nil || *dx != "2021-12-12T01:01:01.000Z" {
		t.Errorf("expected 2021-12-12T01:01:01.000Z; got %s %v", *dx, err)
		return
	}

	if val, err := dx.MarshalText(); err != nil || string(val) != "2021-12-12T01:01:01.000Z" {
		t.Errorf("expected 2021-12-12T01:01:01.000Z; got %s %v", val, err)
		return
	}
	*dx = ""
	if val, err := dx.MarshalText(); err != nil || val != nil {
		t.Errorf("expected null; got %s %v", val, err)
	}

}

func TestDatetime(t *testing.T) {
	var d salesforce.Datetime

	if (&d).Time() != nil {
		t.Errorf("datetime \"\" expected nil Time; got %v", (&d).Time())
		return
	}
	d = "abc"
	if (&d).Time() != nil {
		t.Errorf("datetime %s expected nil Time; got %v", d, (&d).Time())
		return
	}
	d = "2021-01-01T01:01:01.000Z"
	tx := (&d).Time()
	if tx == nil || tx.String() != "2021-01-01 01:01:01 +0000 UTC" {
		t.Errorf("expected 2021-01-01 01:01:01 +0000 UTC; got %v", tx)
	}
	dtfromtm := salesforce.TmToDatetime(nil)
	if dtfromtm != nil {
		t.Errorf("expected nil; got %v", dtfromtm)
		return
	}
	now := time.Now()
	dtfromtm = salesforce.TmToDatetime(&now)
	if dtfromtm == nil || string(*dtfromtm) != now.Format("2006-01-02T15:04:05.000Z0700") {
		t.Errorf("expected %s; got %v", now.Format("2006-01-02T15:04:05.000Z0700"), *dtfromtm)
		return
	}

}

func TestDate_MarshalText(t *testing.T) {
	var d salesforce.Date = ""
	var dx *salesforce.Date
	if err := dx.UnmarshalText([]byte("\"ABCDE\"")); err == nil || err.Error() != "nil pointer" {
		t.Errorf("invalid date expected nil pointer; got %v", err)
		return
	}
	dx = &d

	if err := dx.UnmarshalText([]byte("2021-12-15")); err != nil || *dx != "2021-12-15" {
		t.Errorf("expected 2021-12-15; got %s %v", *dx, err)
		return
	}

	if val, err := dx.MarshalText(); err != nil || string(val) != "2021-12-15" {
		t.Errorf("expected 2021-12-15; got %s %v", val, err)
		return
	}
	*dx = ""
	if val, err := dx.MarshalText(); err != nil || val != nil {
		t.Errorf("expected nil; got %s %v", val, err)
		return
	}
}

func TestDate(t *testing.T) {
	var d salesforce.Date

	if (&d).Time() != nil {
		t.Errorf("date \"\" expected nil Time; got %v", (&d).Time())
		return
	}
	d = "abc"
	if (&d).Time() != nil {
		t.Errorf("date %s expected nil Time; got %v", d, (&d).Time())
		return
	}
	d = "2021-12-12"
	tx := (&d).Time()
	if tx == nil || tx.String() != "2021-12-12 00:00:00 +0000 UTC" {
		t.Errorf("expected 2021-12-12 00:00:00 +0000 UTC; got %v", tx)
	}

	dtfromtm := salesforce.TmToDate(nil)
	if dtfromtm != nil {
		t.Errorf("expected nil; got %v", dtfromtm)
		return
	}
	now := time.Now()
	dtfromtm = salesforce.TmToDate(&now)
	if dtfromtm == nil || string(*dtfromtm) != now.Format("2006-01-02") {
		t.Errorf("expected %s; got %v", now.Format("2006-01-02"), *dtfromtm)
		return
	}
}

func TestTime(t *testing.T) {
	var tm salesforce.Time
	if b, err := tm.MarshalText(); err != nil || string(b) != "null" {
		t.Errorf("expected null from blank; got %s", b)
		return
	}
	tm = "12:01:01"
	if b, err := tm.MarshalText(); err != nil || string(b) != "12:01:01" {
		t.Errorf("expected 12:01:01; got %s", b)
		return
	}
	tx := &tm

	if err := tx.UnmarshalText([]byte("12:28:32")); err != nil || string(*tx) != "12:28:32" {
		t.Errorf("expected 12:28:32; got %v", err)
		return
	}
}

func TestBinary(t *testing.T) {
	if b, err := salesforce.Binary("ABCD").MarshalJSON(); err != nil || string(b) != "\"QUJDRA==\"" {
		t.Errorf("expected QUJDRA==; got %s %v", b, err)
		return
	}
	var buff salesforce.Binary
	if b, err := buff.MarshalJSON(); err != nil || string(b) != "null" {
		t.Errorf("expected null; got %s %v", b, err)
		return
	}
	bx := &buff
	if err := bx.UnmarshalJSON([]byte("=A")); err == nil || !strings.HasPrefix(err.Error(), "invalid character '='") {
		t.Errorf("expected invalid charater '='; got %v", err)
		return
	}
	if err := bx.UnmarshalJSON([]byte("\"QUJDRA==\"")); err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}

}

func TestRecordMap(t *testing.T) {
	var fms = []salesforce.RecordMap{
		{
			"attributes": map[string]interface{}{"type": "sobj000"},
		},
		{
			"attributes": map[string]string{"type": "sobj001"},
		},
		{
			"attributes": salesforce.Attributes{Type: "sobj002"},
		},
		{
			"attributes": &salesforce.Attributes{Type: "sobj003"},
		},
	}
	for i, fm := range fms {
		if fm.SObjectName() != fmt.Sprintf("sobj%03d", i) {
			t.Errorf("expected object name sobj%03d; got %s", i, fm.SObjectName())
		}
	}
	_ = fms[0].WithAttr("ref000")
	ax, ok := fms[0]["attributes"].(*salesforce.Attributes)
	if !ok || ax == nil {
		t.Errorf("expect attributes key = *Attributes")
		return
	}
	if ax.Ref != "ref000" || ax.Type != "sobj000" {
		t.Errorf("expected type=sobj000 and ref=ref000; got %s and %s", ax.Type, ax.Ref)
	}
}

func TestAny_UnmarshalJSON(t *testing.T) {
	salesforce.RegisterSObjectTypes(Contact{}, CustomTable{})

	tests := []struct {
		name      string
		jsonb     string
		errPrefix string
	}{
		{name: "badattr", errPrefix: "attributes decode", jsonb: `{
			"attributes": {
				"type": 5,
				"url": "/services/data/v51.0/sobjects/Contact/0034S000003Quz6QZX"
			},
			"Id": "0034S000003Quz6QZX",
			"IsDeleted": false,
			"External_ID__c": "ABCDEFG",
			"Email__c": "bsmith@example.com",
			"Name__c": "Bob Smith"
		}`},
		{name: "noattr", errPrefix: "attributes not found", jsonb: `{
			"Id": "0034S000003Quz6QZX",
			"IsDeleted": false,
			"External_ID__c": "ABCDEFG",
			"Email__c": "bsmith@example.com",
			"Name__c": "Bob Smith"
		}`},
		{name: "invalidjson", errPrefix: "json: cannot unmarshal number into Go struct field Contact.Id of type string", jsonb: `{
			"attributes": {
				"type": "Contact",
				"url": "/services/data/v51.0/sobjects/Contact/0034S000003Quz6QAC"
			},
			"Id": 5
		}`},
		{name: "Contact", jsonb: `{
			"attributes": {
				"type": "Contact",
				"url": "/services/data/v51.0/sobjects/Contact/0034S000003Quz6QAC"
			},
			"Id": "0034S000003Quz6QAC",
			"IsDeleted": false,
			"AccountId": "0014S000004pwrpQAA",
			"FirstName": "Donald"
		}`},
		{name: "CTable__c", jsonb: `{
			"attributes": {
				"type": "CTable__c",
				"url": "/services/data/v51.0/sobjects/Contact/0034S000003Quz6QZX"
			},
			"Id": "0034S000003Quz6QZX",
			"IsDeleted": false,
			"External_ID__c": "ABCDEFG",
			"Email__c": "bsmith@example.com",
			"Name__c": "Bob Smith"
		}`},
		{name: "CTablex__c", jsonb: `{
			"attributes": {
				"type": "CTablex__c",
				"url": "/services/data/v51.0/sobjects/Contact/0034S000003Quz6QZX"
			},
			"Id": "0034S000003Quz6QZX",
			"IsDeleted": false,
			"External_ID__c": "ABCDEFG",
			"Email__c": "bsmith@example.com",
			"Name__c": "Not Bob Smith"
		}`},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &salesforce.Any{}
			err := a.UnmarshalJSON([]byte(tt.jsonb))
			if tt.errPrefix > "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("%s expected error %q; got %v", tt.name, tt.errPrefix, err)
				}
				return
			}
			if err != nil {
				t.Errorf("%s expected successful unmarshal; got %v", tt.name, err)
				return
			}
			switch rec := a.SObject.(type) {
			case Contact:
				if rec.AccountID != "0014S000004pwrpQAA" {
					t.Errorf("expected Contact.AccountID = 0014S000004pwrpQAA; got %s", rec.AccountID)
				}
			case CustomTable:
				if rec.Name != "Bob Smith" {
					t.Errorf("expected CustomTable.Name = Bob Smith; got %s", rec.Name)
				}
			case salesforce.RecordMap:
				nm, want := rec["Name__c"].(string), "Not Bob Smith"
				if nm != want {
					t.Errorf("expected rec[\"Name__c\"] = Not Bob Smith; got %s", nm)
				}
			default:
				t.Errorf("expected %s; got %s", tt.name, a.SObjectName())
			}
		})
	}
}
