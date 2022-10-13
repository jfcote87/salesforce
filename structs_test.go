// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce_test

import (
	"reflect"
	"testing"

	"github.com/jfcote87/salesforce"
)

func TestNewRecordSlice(t *testing.T) {
	type args struct {
		results interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *salesforce.RecordSlice
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := salesforce.NewRecordSlice(tt.args.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRecordSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRecordSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Account describes the salesforce object Account
type Account struct {
	Attributes              *salesforce.Attributes `json:"attributes,omitempty"`
	AccountID               string                 `json:"Id,omitempty"`                      // [READ-ONLY LOOKUP] id(18)
	Deleted                 bool                   `json:"IsDeleted,omitempty"`               // [READ-ONLY] boolean
	MasterRecordID          string                 `json:"MasterRecordId,omitempty"`          // [READ-ONLY] reference(18)
	MasterRecordIDRel       map[string]string      `json:"MasterRecord,omitempty"`            // update with external id
	AccountName             string                 `json:"Name,omitempty"`                    //  string(255)
	AccountType             string                 `json:"Type,omitempty"`                    //  picklist(255)
	RecordTypeID            string                 `json:"RecordTypeId,omitempty"`            //  reference(18)
	RecordTypeIDRel         map[string]string      `json:"RecordType,omitempty"`              // update with external id
	ParentAccountID         string                 `json:"ParentId,omitempty"`                //  reference(18)
	ParentAccountIDRel      map[string]string      `json:"Parent,omitempty"`                  // update with external id
	BillingStreet           string                 `json:"BillingStreet,omitempty"`           //  textarea(255)
	BillingCity             string                 `json:"BillingCity,omitempty"`             //  string(40)
	BillingStateProvince    string                 `json:"BillingState,omitempty"`            //  string(80)
	BillingZipPostalCode    string                 `json:"BillingPostalCode,omitempty"`       //  string(20)
	BillingCountry          string                 `json:"BillingCountry,omitempty"`          //  string(80)
	BillingLatitude         float64                `json:"BillingLatitude,omitempty"`         //  double
	BillingLongitude        float64                `json:"BillingLongitude,omitempty"`        //  double
	BillingGeocodeAccuracy  string                 `json:"BillingGeocodeAccuracy,omitempty"`  //  picklist(40)
	BillingAddress          *salesforce.Any        `json:"BillingAddress,omitempty"`          // [READ-ONLY] address
	ShippingStreet          string                 `json:"ShippingStreet,omitempty"`          //  textarea(255)
	ShippingCity            string                 `json:"ShippingCity,omitempty"`            //  string(40)
	ShippingStateProvince   string                 `json:"ShippingState,omitempty"`           //  string(80)
	ShippingZipPostalCode   string                 `json:"ShippingPostalCode,omitempty"`      //  string(20)
	ShippingCountry         string                 `json:"ShippingCountry,omitempty"`         //  string(80)
	ShippingLatitude        float64                `json:"ShippingLatitude,omitempty"`        //  double
	ShippingLongitude       float64                `json:"ShippingLongitude,omitempty"`       //  double
	ShippingGeocodeAccuracy string                 `json:"ShippingGeocodeAccuracy,omitempty"` //  picklist(40)
	ShippingAddress         *salesforce.Any        `json:"ShippingAddress,omitempty"`         // [READ-ONLY] address
	AccountPhone            string                 `json:"Phone,omitempty"`                   //  phone(40)
	AccountFax              string                 `json:"Fax,omitempty"`                     //  phone(40)
	AccountNumber           string                 `json:"AccountNumber,omitempty"`           //  string(40)
	Website                 string                 `json:"Website,omitempty"`                 //  url(255)
	VendorID                string                 `json:"Vendor_ID__c,omitempty"`            // [ExternalID LOOKUP] string(16)
}

// SObjectName return rest api name of Account
func (a Account) SObjectName() string {
	return "Account"
}

// WithAttr returns a new Account with attributes of Type="Account"
// and Ref=ref
func (a Account) WithAttr(ref string) salesforce.SObject {
	a.Attributes = &salesforce.Attributes{Type: "Account", Ref: ref}
	return a
}

// NotSObject used to validate sobject tests
type NotSObject struct {
	A string
}

// Contact describes the salesforce object Contact
type Contact struct {
	Attributes             *salesforce.Attributes `json:"attributes,omitempty"`
	ContactID              string                 `json:"Id,omitempty"`             // [READ-ONLY LOOKUP] id(18)
	IsDeleted              bool                   `json:"IsDeleted,omitempty"`      // [READ-ONLY] boolean
	MasterRecordID         string                 `json:"MasterRecordId,omitempty"` // [READ-ONLY] reference(18)
	AccountID              string                 `json:"AccountId,omitempty"`      // reference(18)
	AccountIDRel           map[string]interface{} `json:"Account,omitempty"`        // update with external id
	LastName               string                 `json:"LastName,omitempty"`       // string(80)
	FirstName              string                 `json:"FirstName,omitempty"`      // string(40)
	Salutation             string                 `json:"Salutation,omitempty"`
	MiddleName             string                 `json:"MiddleName,omitempty"`             // string(40)
	Name                   string                 `json:"Name,omitempty"`                   // [READ-ONLY] string(121)
	RecordTypeID           string                 `json:"RecordTypeId,omitempty"`           // reference(18)
	RecordTypeIDRel        map[string]interface{} `json:"RecordType,omitempty"`             // update with external id
	OtherStreet            string                 `json:"OtherStreet,omitempty"`            // textarea(255)
	OtherCity              string                 `json:"OtherCity,omitempty"`              // string(40)
	OtherState             string                 `json:"OtherState,omitempty"`             // string(80)
	OtherPostalCode        string                 `json:"OtherPostalCode,omitempty"`        // string(20)
	OtherCountry           string                 `json:"OtherCountry,omitempty"`           // string(80)
	OtherLatitude          float64                `json:"OtherLatitude,omitempty"`          // double
	OtherLongitude         float64                `json:"OtherLongitude,omitempty"`         // double
	OtherGeocodeAccuracy   string                 `json:"OtherGeocodeAccuracy,omitempty"`   // picklist(40)
	OtherAddress           *salesforce.Any        `json:"OtherAddress,omitempty"`           // [READ-ONLY] address
	MailingStreet          string                 `json:"MailingStreet,omitempty"`          // textarea(255)
	MailingCity            string                 `json:"MailingCity,omitempty"`            // string(40)
	MailingState           string                 `json:"MailingState,omitempty"`           // string(80)
	MailingPostalCode      string                 `json:"MailingPostalCode,omitempty"`      // string(20)
	MailingCountry         string                 `json:"MailingCountry,omitempty"`         // string(80)
	MailingLatitude        float64                `json:"MailingLatitude,omitempty"`        // double
	MailingLongitude       float64                `json:"MailingLongitude,omitempty"`       // double
	MailingGeocodeAccuracy string                 `json:"MailingGeocodeAccuracy,omitempty"` // picklist(40)
	MailingAddress         *salesforce.Any        `json:"MailingAddress,omitempty"`         // [READ-ONLY] address
	Phone                  string                 `json:"Phone,omitempty"`                  // phone(40)
	Fax                    string                 `json:"Fax,omitempty"`                    // phone(40)
	MobilePhone            string                 `json:"MobilePhone,omitempty"`            // phone(40)
	HomePhone              string                 `json:"HomePhone,omitempty"`              // phone(40)
	DoNotCall              bool                   `json:"DoNotCall,omitempty"`              // boolean
	ExternalPID            string                 `json:"PID__c,omitempty"`                 // [READ-ONLY CALCULATED] string(1300)
}

// SObjectName return rest api name of Contact
func (c Contact) SObjectName() string {
	return "Contact"
}

// WithAttr returns a new Contact with attributes of Type="Contact"
// and Ref=ref
func (c Contact) WithAttr(ref string) salesforce.SObject {
	c.Attributes = &salesforce.Attributes{Type: "Contact", Ref: ref}
	return c
}

// CustomTable describes custom salesforce object CTable__c
type CustomTable struct {
	Attributes *salesforce.Attributes `json:"attributes,omitempty"`
	ID         string                 `json:"Id,omitempty"`        // [READ-ONLY LOOKUP] id(18)
	Deleted    bool                   `json:"IsDeleted,omitempty"` // [READ-ONLY] boolean
	Name       string                 `json:"Name__c,omitempty"`   //  string(80)
	Email      string                 `json:"Email__c,omitempty"`  //  string(40)
	ExternalID string                 `json:"External_ID__c,omitempty"`
}

// SObjectName return rest api name of Contact
func (c CustomTable) SObjectName() string {
	return "CTable__c"
}

// WithAttr returns a new Contact with attributes of Type="Contact"
// and Ref=ref
func (c CustomTable) WithAttr(ref string) salesforce.SObject {
	c.Attributes = &salesforce.Attributes{Type: "CTable__c", Ref: ref}
	return c
}

func TestAddress(t *testing.T) {
	addr := salesforce.Address{
		GeocodeAccuracy: "a",
		City:            "Denver",
		CountryCode:     "US",
		StateCode:       "CO",
		PostalCode:      "80203",
		Street:          "5508 Friendship",
		Longitude:       -105.27055,
		Latitude:        40.01499,
	}
	mx := addr.ToMap("a1", true)
	ax := salesforce.ToAddress("a1", mx)
	if ax.GeocodeAccuracy != addr.GeocodeAccuracy ||
		ax.City != addr.City ||
		ax.CountryCode != addr.CountryCode ||
		ax.StateCode != addr.StateCode ||
		ax.PostalCode != addr.PostalCode ||
		ax.Street != addr.Street ||
		ax.Latitude != addr.Latitude ||
		ax.Longitude != addr.Longitude {
		t.Errorf("address values do not match")
	}

}
