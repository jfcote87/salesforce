// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce

import (
	"reflect"
)

const defaultBatchSize = 2000

// Field defines a field of an sobject
// https://developer.salesforce.com/docs/atlas.en-us.api.meta/api/sforce_api_calls_describesobjects_describesobjectresult.htm
// Scroll down for Field definition
type Field struct {
	Aggregatable                 bool            `json:"aggregatable,omitempty"`
	AiPredictionField            bool            `json:"aiPredictionField,omitempty"`
	AutoNumber                   bool            `json:"autoNumber,omitempty"`
	ByteLength                   int             `json:"byteLength,omitempty"`
	Calculated                   bool            `json:"calculated,omitempty"`
	CalculatedFormula            interface{}     `json:"calculatedFormula,omitempty"`
	CascadeDelete                bool            `json:"cascadeDelete,omitempty"`
	CaseSensitive                bool            `json:"caseSensitive,omitempty"`
	CompoundFieldName            interface{}     `json:"compoundFieldName,omitempty"`
	ControllerName               interface{}     `json:"controllerName,omitempty"`
	Createable                   bool            `json:"createable,omitempty"`
	Custom                       bool            `json:"custom,omitempty"`
	DefaultedOnCreate            bool            `json:"defaultedOnCreate,omitempty"`
	DefaultValueFormula          interface{}     `json:"defaultValueFormula,omitempty"`
	DefaultValue                 interface{}     `json:"defaultValue,omitempty"`
	DependentPicklist            bool            `json:"dependentPicklist,omitempty"`
	DeprecatedAndHidden          bool            `json:"deprecatedAndHidden,omitempty"`
	Digits                       int             `json:"digits,omitempty"`
	DisplayLocationInDecimal     bool            `json:"displayLocationInDecimal,omitempty"`
	Encrypted                    bool            `json:"encrypted,omitempty"`
	ExternalID                   bool            `json:"externalId,omitempty"`
	ExtraTypeInfo                interface{}     `json:"extraTypeInfo,omitempty"`
	Filterable                   bool            `json:"filterable,omitempty"`
	FilteredLookupInfo           interface{}     `json:"filteredLookupInfo,omitempty"`
	FormulaTreatNullNumberAsZero bool            `json:"formulaTreatNullNumberAsZero,omitempty"`
	Groupable                    bool            `json:"groupable,omitempty"`
	HighScaleNumber              bool            `json:"highScaleNumber,omitempty"`
	HTMLFormatted                bool            `json:"htmlFormatted,omitempty"`
	IDLookup                     bool            `json:"idLookup,omitempty"`
	InlineHelpText               string          `json:"inlineHelpText,omitempty"`
	Label                        string          `json:"label,omitempty"`
	Length                       int             `json:"length,omitempty"`
	Mask                         string          `json:"mask,omitempty"`
	MaskType                     string          `json:"maskType,omitempty"`
	NameField                    bool            `json:"nameField,omitempty"`
	NamePointing                 bool            `json:"namePointing,omitempty"`
	Name                         string          `json:"name,omitempty"`
	Nillable                     bool            `json:"nillable,omitempty"`
	Permissionable               bool            `json:"permissionable,omitempty"`
	PicklistValues               []PickListValue `json:"picklistValues,omitempty"`
	PolymorphicForeignKey        bool            `json:"polymorphicForeignKey,omitempty"`
	Precision                    int             `json:"precision,omitempty"`
	QueryByDistance              bool            `json:"queryByDistance,omitempty"`
	ReferenceTargetField         string          `json:"referenceTargetField,omitempty"`
	ReferenceTo                  []string        `json:"referenceTo,omitempty"`
	RelationshipName             string          `json:"relationshipName,omitempty"`
	RelationshipOrder            int             `json:"relationshipOrder,omitempty"`
	RestrictedDelete             bool            `json:"restrictedDelete,omitempty"`
	RestrictedPicklist           bool            `json:"restrictedPicklist,omitempty"`
	Scale                        int             `json:"scale,omitempty"`
	SearchPrefilterable          bool            `json:"searchPrefilterable,omitempty"`
	SoapType                     string          `json:"soapType,omitempty"`
	Sortable                     bool            `json:"sortable,omitempty"`
	Type                         string          `json:"type,omitempty"`
	Unique                       bool            `json:"unique,omitempty"`
	Updateable                   bool            `json:"updateable,omitempty"`
	WriteRequiresMasterRead      bool            `json:"writeRequiresMasterRead,omitempty"`
}

// ChildRef describes sobject details
// https://developer.salesforce.com/docs/atlas.en-us.api.meta/api/sforce_api_calls_describesobjects_describesobjectresult.htm
// Scroll down for ChildRef definition
type ChildRef struct {
	CascadeDelete       bool          `json:"cascadeDelete,omitempty"`
	ChildSObject        interface{}   `json:"childSObject,omitempty"`
	DeprecatedAndHidden bool          `json:"deprecatedAndHidden,omitempty"`
	Field               string        `json:"field,omitempty"`
	JunctionIDListNames []string      `json:"junctionIdListNames,omitempty"`
	JunctionReferenceTo []interface{} `json:"junctionReferenceTo,omitempty"`
	RelationshipName    *string       `json:"relationshipName,omitempty"`
	RestrictedDelete    bool          `json:"restrictedDelete,omitempty"`
}

// Scope describes an sobject scope
type Scope struct {
	Label string `json:"label,omitempty"`
	Name  string `json:"name,omitempty"`
}

// Links lists all sobject links
type Links struct {
	ApprovalLayouts  string `json:"approvalLayouts,omitempty"`
	CompactLayouts   string `json:"compactLayouts,omitempty"`
	DefaultValues    string `json:"defaultValues,omitempty"`
	Describe         string `json:"describe,omitempty"`
	Layouts          string `json:"layouts,omitempty"`
	Listviews        string `json:"listviews,omitempty"`
	QuickActions     string `json:"quickActions,omitempty"`
	RowTemplate      string `json:"rowTemplate,omitempty"`
	Sobject          string `json:"sobject,omitempty"`
	UIDetailTemplate string `json:"uiDetailTemplate,omitempty"`
	UIEditTemplate   string `json:"uiEditTemplate,omitempty"`
	UINewRecord      string `json:"uiNewRecord,omitempty"`
}

// RecordTypeInfo for sobject
type RecordTypeInfo struct {
	Active                   bool              `json:"active,omitempty"`
	Available                bool              `json:"available,omitempty"`
	DefaultRecordTypeMapping bool              `json:"defaultRecordTypeMapping,omitempty"`
	DeveloperName            string            `json:"developerName,omitempty"`
	Layout                   map[string]string `json:"layout,omitempty"`
	Master                   bool              `json:"master,omitempty"`
	Name                     string            `json:"name,omitempty"`
	RecordTypeID             string            `json:"recordTypeId,omitempty"`
	Urls                     Links             `json:"urls,omitempty"`
}

// PickListValue describes l
type PickListValue struct {
	Active       bool        `json:"active,omitempty"`
	DefaultValue bool        `json:"defaultValue,omitempty"`
	Label        string      `json:"label,omitempty"`
	ValidFor     interface{} `json:"validFor,omitempty"`
	Value        string      `json:"value,omitempty"`
}

// ActionOverride provides details about an action that replaces the
// default action pages for an object. For example, an object could be
// configured to replace the new/create page with a custom page.
// https://developer.salesforce.com/docs/atlas.en-us.api.meta/api/sforce_api_calls_describesobjects_describesobjectresult.htm
type ActionOverride struct {
	FormFactor           string `json:"formFactor,omitempty"`
	IsIsAvailableInTouch bool   `json:"isAvailableInTouch,omitempty"`
	Name                 string `json:"name,omitempty"`
	PageID               string `json:"pageID,omitempty"`
	URL                  string `json:"url,omitempty"`
}

// SObjectDefinition describes a salesforce SObject
// https://developer.salesforce.com/docs/atlas.en-us.api.meta/api/sforce_api_calls_describesobjects_describesobjectresult.htm
type SObjectDefinition struct {
	Activateable          bool             `json:"activateable,omitempty"`
	ActionOverrides       []ActionOverride `json:"actionOverrides"`
	AssociateEntityType   *string          `json:"associateEntityType,omitempty"`
	AssociateParentEntity string           `json:"associateParentEntity,omitempty"`
	ChildRelationships    []ChildRef       `json:"childRelationships,omitempty"`
	CompactLayoutable     bool             `json:"compactLayoutable,omitempty"`
	Createable            bool             `json:"createable,omitempty"`
	Custom                bool             `json:"custom,omitempty"`
	CustomSetting         bool             `json:"customSetting,omitempty"`
	DeepCloneable         bool             `json:"deepCloneable,omitempty"`
	DefaultImplementation interface{}      `json:"defaultImplementation,omitempty"`
	Deletable             bool             `json:"deletable,omitempty"`
	DeprecatedAndHidden   bool             `json:"deprecatedAndHidden,omitempty"`
	ExtendedBy            interface{}      `json:"extendedBy,omitempty"`
	ExtendsInterfaces     interface{}      `json:"extendsInterfaces,omitempty"`
	FeedEnabled           bool             `json:"feedEnabled,omitempty"`
	Fields                []Field          `json:"fields,omitempty"`
	HasSubtypes           bool             `json:"hasSubtypes,omitempty"`
	ImplementedBy         interface{}      `json:"implementedBy,omitempty"`
	ImplementsInterfaces  interface{}      `json:"implementsInterfaces,omitempty"`
	IsInterface           bool             `json:"isInterface,omitempty"`
	IsSubtype             bool             `json:"isSubtype,omitempty"`
	KeyPrefix             string           `json:"keyPrefix,omitempty"`
	LabelPlural           string           `json:"labelPlural,omitempty"`
	Label                 string           `json:"label,omitempty"`
	Layoutable            bool             `json:"layoutable,omitempty"`
	Listviewable          interface{}      `json:"listviewable,omitempty"`
	LookupLayoutable      interface{}      `json:"lookupLayoutable,omitempty"`
	Mergeable             bool             `json:"mergeable,omitempty"`
	MruEnabled            bool             `json:"mruEnabled,omitempty"`
	NamedLayoutInfos      []interface{}    `json:"namedLayoutInfos,omitempty"`
	Name                  string           `json:"name,omitempty"`
	NetworkScopeFieldName interface{}      `json:"networkScopeFieldName,omitempty"`
	Queryable             bool             `json:"queryable,omitempty"`
	RecordTypeInfos       []RecordTypeInfo `json:"recordTypeInfos,omitempty"`
	Replicateable         bool             `json:"replicateable,omitempty"`
	Retrieveable          bool             `json:"retrieveable,omitempty"`
	Searchable            bool             `json:"searchable,omitempty"`
	SearchLayoutable      bool             `json:"searchLayoutable,omitempty"`
	SobjectDescribeOption string           `json:"sobjectDescribeOption,omitempty"`
	SupportedScopes       []Scope          `json:"supportedScopes,omitempty"`
	Triggerable           bool             `json:"triggerable,omitempty"`
	Undeletable           bool             `json:"undeletable,omitempty"`
	Updateable            bool             `json:"updateable,omitempty"`
	Urls                  Links            `json:"urls,omitempty"`
}

// Attributes data returned with each query record
type Attributes struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
	Ref  string `json:"referenceId,omitempty"`
}

// JobDefinition is the initialization data for a new bulk job
type JobDefinition struct {
	ExternalIDFieldName string `json:"externalIdFieldName,omitempty"`
	Object              string `json:"object,omitempty"`
	Operation           string `json:"operation,omitempty"`
	ConcurrencyMode     string `json:"concurrencyMode,omitempty"`
	ContentType         string `json:"contentType,omitempty"`
	LineEnding          string `json:"lineEnding,omitempty"`
	ColumnDelimiter     string `json:"columnDelimiter,omitempty"`
	AssignmentRuleID    string `json:"assignmentRuleId,omitempty"`
}

// Job contains current state of a job and is returned from the
// CreateJob, CloseJob, GetJob and AbortJob methods
type Job struct {
	APIVersion             float64 `json:"apiVersion,omitempty"`
	AssignmentRuleID       string  `json:"assignmentRuleId,omitempty"`
	ColumnDelimiter        string  `json:"columnDelimiter,omitempty"`
	ConcurrencyMode        string  `json:"concurrencyMode,omitempty"`
	ContentType            string  `json:"contentType,omitempty"`
	ContentURL             string  `json:"contentURL,omitempty"`
	CreatedByID            string  `json:"createdById,omitempty"`
	CreatedDate            string  `json:"createdDate,omitempty"`
	ExternalIDFieldName    string  `json:"externalIdFieldName,omitempty"`
	ID                     string  `json:"id,omitempty"`
	JobType                string  `json:"jobType,omitempty"`
	LineEnding             string  `json:"lineEnding,omitempty"`
	NumberRecordsFailed    int     `json:"numberRecordsFailed"`
	NumberRecordsProcessed int     `json:"numberRecordsProcessed"`
	Object                 string  `json:"object,omitempty"`
	Operation              string  `json:"operation,omitempty"`
	State                  string  `json:"state,omitempty"`
	SystemModstamp         string  `json:"systemModstamp,omitempty"`
}

// JobList returns all job status
type JobList struct {
	Done           bool   `json:"done,omitempty"`
	Records        []Job  `json:"records,omitempty"`
	NextRecordsURL string `json:"nextRecordsUrl,omitempty"`
}

// Address is describes structure of the Address type.  Field names
// differ depending on the object.  Convert to and from map[string]interface{}
// with specific routines
type Address struct {
	GeocodeAccuracy string  `json:"geocodeAccuracy,omitempty"` //	Accuracy level of the geocode for the address. For example, this field is known as MailingGeocodeAccuracy on Contact.
	City            string  `json:"city,omitempty"`            // The city detail for the address. For example, this field is known as MailingCity on Contact.
	CountryCode     string  `json:"country,omitempty"`         // The ISO country code for the address. For example, this field is known as MailingCountryCode on Contact. CountryCode is always available on compound address fields, whether or not state and country picklists are enabled in your organization.
	Latitude        float64 `json:"latitude,omitempty"`        // Used with Longitude to specify the precise geolocation of the address. For example, this field is known as MailingLatitude on Contact.
	Longitude       float64 `json:"longitude,omitempty"`       // Used with Latitude to specify the precise geolocation of the address. For example, this field is known as MailingLongitude on Contact.
	PostalCode      string  `json:"postalCode,omitempty"`      // The postal code for the address. For example, this field is known as MailingPostalCode on Contact.
	StateCode       string  `json:"state,omitempty"`           // The ISO state code for the address. For example, this field is known as MailingStateCode on Contact. StateCode is always available on compound address fields, whether or not state and country picklists are enabled in your organization.
	Street          string  `json:"street,omitempty"`          // textarea
}

// ToMap creates a map that may be used to update address type. nm
// is the prefix for the field names.
func (a Address) ToMap(nm string, omitempty bool) map[string]interface{} {
	m := make(map[string]interface{})
	val := reflect.ValueOf(a)
	ty := reflect.TypeOf(a)
	for i := 0; i < val.NumField(); i++ {
		if !omitempty || !val.Field(i).IsZero() {
			fieldNm := ty.Field(i).Name
			m[nm+fieldNm] = val.Field(i).Interface()
		}
	}
	return m
}

// ToAddress converts a map[string]interface{} to an Address
// value.  The prefix parameter allows mapping of map fields
// to Address fields.
func ToAddress(prefix string, ix map[string]interface{}) *Address {
	return &Address{
		GeocodeAccuracy: stringFromInterface(ix[prefix+"GeocodeAccuracy"]),
		City:            stringFromInterface(ix[prefix+"City"]),
		//Country:         stringFromInterface(ix[prefix+"Country"]),
		CountryCode: stringFromInterface(ix[prefix+"CountryCode"]),
		//State:           stringFromInterface(ix[prefix+"State"]),
		StateCode:  stringFromInterface(ix[prefix+"StateCode"]),
		PostalCode: stringFromInterface(ix[prefix+"PostalCode"]),
		Street:     stringFromInterface(ix[prefix+"Street"]),
		Longitude:  float64FromInterface(ix[prefix+"Longitude"]),
		Latitude:   float64FromInterface(ix[prefix+"Latitude"]),
	}

}

func stringFromInterface(ix interface{}) string {
	s, _ := ix.(string)
	return s
}

func float64FromInterface(ix interface{}) float64 {
	s, _ := ix.(float64)
	return s
}
