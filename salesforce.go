// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package salesforce implements data access, creation and updating
// routines for the Salesforce Rest API
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm
package salesforce // import github.org/jfcote87/salesforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// QueryResponse is the base response for a query
type QueryResponse struct {
	TotalSize      int          `json:"totalSize,omitempty"`
	Done           bool         `json:"done,omitempty"`
	NextRecordsURL string       `json:"nextRecordsUrl,omitempty"`
	Records        *RecordSlice `json:"records"`
}

// RecordSlice wraps pointer to the result slice allowing
// custom unmarshaling that adds rows through multiple
// next record calls
type RecordSlice struct {
	resultsVal  reflect.Value
	resultsType reflect.Type
}

func (rs *RecordSlice) rows() int {
	return rs.resultsVal.Len()
}

// slice resets the value of resultsVal to resultsVal.Interface()[i:j]
func (rs *RecordSlice) slice(i, j int) {
	rs.resultsVal.Set(rs.resultsVal.Slice(i, j))
}

// NewRecordSlice creates a RecordSlice pointer based upon *[]<struct> of the results
// parameter; error created when result is an invalid type
func NewRecordSlice(results interface{}) (*RecordSlice, error) {
	ptr := reflect.ValueOf(results)
	pType := ptr.Type()
	if pType.Kind() != reflect.Ptr || pType.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected *[]<struct>; got %v", ptr.Type())
	}
	slice := ptr.Elem()
	return &RecordSlice{
		resultsVal:  slice,
		resultsType: slice.Type(),
	}, nil
}

// UnmarshalJSON creates a temporary slice for unmarshaling a call.  This temp
// slice is then appended to the resultsVal
func (rs *RecordSlice) UnmarshalJSON(b []byte) error {
	if rs == nil || reflect.ValueOf(rs.resultsVal).IsZero() {
		return fmt.Errorf("uninitialized QueryResult")
	}
	tempSlice := reflect.New(rs.resultsType)
	// initialize tempSlice
	tempSlice.Elem().Set(reflect.AppendSlice(reflect.MakeSlice(rs.resultsType, 0, defaultBatchSize), tempSlice.Elem()))
	if err := json.Unmarshal(b, tempSlice.Interface()); err != nil {
		return err
	}

	rs.resultsVal.Set(reflect.AppendSlice(rs.resultsVal, tempSlice.Elem()))
	return nil
}

// MarshalJSON marshals the value in resultsVal
func (rs RecordSlice) MarshalJSON() ([]byte, error) {
	if !rs.resultsVal.IsNil() && rs.resultsVal.CanInterface() {
		return json.Marshal(rs.resultsVal.Interface())
	}
	return nil, nil
}

const defaultDatetimeFormat = "2006-01-02T15:04:05.000Z0700"
const defaultDateFormat = "2006-01-02"

// Time converts the string to a time.Time value
func (d *Datetime) Time() *time.Time {
	if d == nil || *d == "" {
		return nil
	}
	tm, err := time.Parse(defaultDatetimeFormat, string(*d))
	if err != nil || tm.IsZero() {
		return nil
	}
	return &tm

}

// Time converts the string to a time.Time value
func (d *Date) Time() *time.Time {
	if d == nil || *d == "" {
		return nil
	}
	tm, err := time.Parse(defaultDateFormat, string(*d))
	if err != nil || tm.IsZero() {
		return nil
	}
	return &tm

}

// TmToDate converts a time.Time to a Date pointer with zero time value
func TmToDate(tm *time.Time) *Date {
	if tm != nil && !tm.IsZero() {
		dt := Date(tm.Format("2006-01-02"))
		return &dt
	}
	return nil
}

// TmToDatetime converts a time.Time to DateTime
func TmToDatetime(tm *time.Time) *Datetime {
	if tm != nil && !tm.IsZero() {
		dt := Datetime(tm.Format("2006-01-02T15:04:05.000Z0700"))
		return &dt
	}
	return nil
}

// Date handles the json marshaling and unmarshaling of SF date type which
// is a string of format yyyy-mm-dd
type Date string

// Datetime  handles the json marshaling and unmarshaling of SF datetime type
// which a string of iso 8061 format yyyy-mm-ddThh:mm:ss.sss+0000
type Datetime string

// Time handles the json marshaling and unmarshaling of SF time type
type Time string

// MarshalText handles outputting json of date.  Empty value
// outputs null, a nil ptr is omitted with omitempty.
func (d *Datetime) MarshalText() ([]byte, error) {
	if d != nil && *d > "" {
		return []byte(*d), nil
	}
	return nil, nil
}

// UnmarshalText does null handling during json decode,
func (d *Datetime) UnmarshalText(b []byte) error {
	if d == nil {
		return fmt.Errorf("nil pointer")
	}
	*d = Datetime(b)
	return nil
}

// Display allows different formatting of *Date
// and displays nils and empty strings
func (d *Date) Display(format string) string {
	tm := d.Time()
	if tm == nil {
		return ""
	}
	if format == "" {
		format = "2006-01-02"
	}
	return tm.Format(format)
}

// MarshalText handles outputting json of date.  Empty value
// outputs null, a nil ptr is omitted with omitempty.
func (d Date) MarshalText() ([]byte, error) {
	if d > "" {
		return []byte(d), nil
	}
	return nil, nil
}

// UnmarshalText does null handling during json decode,
func (d *Date) UnmarshalText(b []byte) error {
	if d == nil {
		return fmt.Errorf("nil pointer")
	}
	*d = Date(b)
	return nil
}

// MarshalText handles outputting json of time.  Empty value
// outputs null, a nil ptr is omitted with omitempty.
func (t Time) MarshalText() ([]byte, error) {
	if t > "" {
		return []byte(t), nil
	}
	return []byte("null"), nil
}

// UnmarshalText does null handling during json decode,
func (t *Time) UnmarshalText(b []byte) error {
	if t == nil {
		return fmt.Errorf("nil pointer")
	}
	*t = Time(b)
	return nil
}

// Binary handles base64Binary type
type Binary []byte

// MarshalJSON handles outputting json of base64Binary.  Empty value
// outputs null, a nil ptr is omitted with omitempty.
func (b Binary) MarshalJSON() ([]byte, error) {
	if len(b) > 0 {
		return json.Marshal([]byte(b))
	}
	return []byte("null"), nil
}

// UnmarshalJSON does null handling during json decode,
func (b *Binary) UnmarshalJSON(buff []byte) error {
	var bx []byte
	if err := json.Unmarshal(buff, &bx); err != nil {
		return err
	}
	*b = bx
	return nil
}

// NullValue represents a sent or received null in json
type NullValue struct{}

// RecordMap is created during an Any.JSONUnmarshal when
// the record type is not registered
type RecordMap map[string]interface{}

// SObjectName returns the attributes type value
func (m RecordMap) SObjectName() string {
	if m != nil {
		switch ix := m["attributes"].(type) {
		case map[string]interface{}:
			nm, _ := ix["type"].(string)
			return nm
		case map[string]string:
			return ix["type"]
		case Attributes:
			return ix.Type
		case *Attributes:
			return ix.Type
		}
	}
	return ""
}

// WithAttr set the attributes value
func (m RecordMap) WithAttr(ref string) SObject {
	if m != nil {
		m["attributes"] = &Attributes{
			Type: m.SObjectName(),
			Ref:  ref,
		}
	}
	return m
}

// Any is used to unmarshal an SObject json for undetermined objects.
type Any struct {
	SObject
}

// UnmarshalJSON uses the attributes data to determine the SObject type
// the registered structs.  SObject
func (a *Any) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	var attr *Attributes
	for {
		t, err := dec.Token()
		if err == nil {
			nm, _ := t.(string)
			if nm == "attributes" {
				if err = dec.Decode(&attr); err != nil {
					return fmt.Errorf("attributes decode %w", err)
				}
				break
			}
			continue
		}
		return fmt.Errorf("attributes not found %v", err)
	}
	sobjPtrVal := sobjCatalog.getNewValue(attr.Type)

	if err := json.Unmarshal(b, sobjPtrVal.Interface()); err != nil {
		return err
	}
	a.SObject = sobjPtrVal.Elem().Interface().(SObject)
	return nil
}

var mapSObjectStructs = make(map[string]reflect.Type)

var sobjCatalog = &catalog{sobjects: make(map[string]reflect.Type)}

// RegisterSObjectTypes catalogs the type of the SObject structs.  The Any
// type uses these registrations to unmarshal a salesforce response
// into the appropriate type.
func RegisterSObjectTypes(sobjs ...SObject) {
	for _, o := range sobjs {
		sobjCatalog.sobjects[o.SObjectName()] = reflect.TypeOf(o)
	}
}

type catalog struct {
	sobjects map[string]reflect.Type
	m        sync.Mutex
}

func (c *catalog) getNewValue(name string) reflect.Value {
	c.m.Lock()
	defer c.m.Unlock()
	ty, ok := c.sobjects[name]
	if ok {
		return reflect.New(ty)
	}
	var mapValues RecordMap
	return reflect.New(reflect.TypeOf(mapValues))
}

// SObject is a struct used for Create, Update and Delete operations
type SObject interface {
	SObjectName() string
	WithAttr(string) SObject
}

// Error is the error response for most calls
type Error struct {
	StatusCode string   `json:"statusCode,omitempty"`
	Message    string   `json:"message,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

// LogError used to report individual record errors
type LogError struct {
	Index      int     `json:"index"`
	ExternalID string  `json:"external_id,omitempty"`
	Errors     []Error `json:"errors,omitempty"`
}
