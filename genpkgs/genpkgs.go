// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package genpkgs is a library for creating packages to describe the queryable
// sobjects in a salesforce instance.  Using the Config object, a developer can
// create a single or multiple packages containg struct definitions for the
// desired objects.
package genpkgs // import github.com/jfcote87/salesforce/genpkgs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/jfcote87/salesforce"

	"github.com/mgechev/revive/lint"
)

var defaultTemplate *template.Template

func init() {
	defaultTemplate = template.Must(template.New("defs").Parse(defaultTemplateSource))
}

var defaulttypeMap = map[string]string{
	"ChangeEventHeader":                "*salesforce.Any",
	"StringList":                       "string",
	"tns:ID":                           "string",
	"urn:JunctionIdListNames":          "*salesforce.Any",
	"urn:RecordTypesSupported":         "*salesforce.Any",
	"urn:RelationshipReferenceTo":      "*salesforce.Any",
	"urn:SearchLayoutButtonsDisplayed": "*salesforce.Any",
	"urn:SearchLayoutFieldsDisplayed":  "*salesforce.Any",
	"urn:address":                      "*salesforce.Address",
	"urn:location":                     "*salesforce.Any",
	"xsd:anyType":                      "*salesforce.Any",
	"xsd:base64Binary":                 "*salesforce.Binary",
	"xsd:boolean":                      "bool",
	"xsd:date":                         "*salesforce.Date",
	"xsd:dateTime":                     "*salesforce.Datetime",
	"xsd:double":                       "float64",
	"xsd:int":                          "int",
	"xsd:long":                         "int",
	"xsd:string":                       "string",
	"xsd:time":                         "*salesforce.Time",
}

var numberOfGoRoutines = 8

// Config holds parameters for code generation
type Config struct {
	SoapTypes                   map[string]string    `json:"soap_type,omitempty"`                      // salesforce soap type to go type map. Used to replace or insert to default mappings
	SkipObjects                 []string             `json:"skip,omitempty"`                           // objects to exclude from generated package
	StructOverrides             map[string]*Override `json:"struct_overrides,omitempty"`               // map of structs with override
	SkipRelationshipGlobal      map[string]bool      `json:"skip_relationship_global,omitempty"`       // relationshipnames to skip in every object
	Packages                    []Parameters         `json:"packages,omitempty"`                       // list of Packages to create
	IncludeCodeGeneratedComment bool                 `json:"include_code_generated_comment,omitempty"` // add Code generated .* DO NOT EDIT.$

}

// MakeTemplateData generates a slice of Templates
func (cfg *Config) MakeTemplateData(ctx context.Context, sv *salesforce.Service) ([]*TemplateData, error) {
	job, err := cfg.ReadSObjectDescriptions(ctx, sv)
	if err != nil {
		return nil, err
	}

	var results = make([]*TemplateData, len(cfg.Packages))
	// loop thru packages and generate data for template
	for idx := range cfg.Packages {
		pkg := &job.Packages[idx]
		if len(job.StructMap[pkg]) == 0 {
			log.Printf("Package %s has no objects", pkg.Name)
		}

		td := job.TemplateData(pkg)
		if td == nil {
			log.Printf("Package (%s) record not found", pkg.Name)
		}
		results[idx] = td
	}
	return results, nil
}

// ErrorList contains a slice of errors.
type ErrorList []error

func (el ErrorList) Error() string {
	if len(el) > 0 {
		var msg []byte
		for i, e := range el {
			if i > 0 {
				msg = append(msg, '\n')
			}
			msg = append(msg, []byte(e.Error())...)
		}
		return string(msg)
	}
	return ""
}

type jobMaps struct {
	includeRegexpMap map[*Parameters]*regexp.Regexp
	replaceRegexpMap map[*Parameters]*regexp.Regexp
	replaceTextMap   map[*Parameters]string
	skipMap          map[string]bool
	typeMap          sfTypeMap
}

func (cfg *Config) getMaps() (*jobMaps, error) {
	// compile regular expressions
	jm := &jobMaps{
		includeRegexpMap: make(map[*Parameters]*regexp.Regexp),
		replaceRegexpMap: make(map[*Parameters]*regexp.Regexp),
		replaceTextMap:   make(map[*Parameters]string),
		skipMap:          make(map[string]bool),
		typeMap:          make(sfTypeMap),
	}

	for idx := range cfg.Packages {
		pkg := &cfg.Packages[idx]
		includeRegexp, replaceRegexp, replaceText, err := pkg.Validate()
		if err != nil {
			return nil, fmt.Errorf("package[%d] (name=%s) validation failed %v", idx, pkg.Name, err)
		}
		if includeRegexp != nil {
			jm.includeRegexpMap[pkg] = includeRegexp
		}
		if replaceRegexp != nil {
			jm.replaceRegexpMap[pkg] = replaceRegexp
			jm.replaceTextMap[pkg] = replaceText
		}
	}
	// lists to maps
	for _, s := range cfg.SkipObjects {
		jm.skipMap[s] = true
	}
	for k, v := range defaulttypeMap {
		jm.typeMap[k] = v
	}
	// overwrite/add from config
	for k, v := range cfg.SoapTypes {
		jm.typeMap[k] = v
	}
	return jm, nil

}

// CreateJob initializes a Job struct with the salesforce instance's list of SObjects,
// and creates maps for caching related data
func (cfg *Config) CreateJob(ctx context.Context, sv *salesforce.Service) (*Job, error) {
	jm, err := cfg.getMaps()
	if err != nil {
		return nil, err
	}

	// read objects from salesforce instance
	objs, err := sv.ObjectList(ctx)
	if err != nil {
		return nil, fmt.Errorf("object list failed: %w", err)
	}

	var objMap = make(map[string]salesforce.SObjectDefinition, len(objs))
	for _, o := range objs {
		if !jm.skipMap[o.Name] &&
			(o.Searchable || o.Retrieveable || o.Updateable || o.Queryable || o.Createable) {
			objMap[o.Name] = o
		}
	}
	structMap := make(map[*Parameters][]Struct)
	for i := range cfg.Packages {
		structMap[&cfg.Packages[i]] = make([]Struct, 0)
	}
	return &Job{
		Config:       cfg,
		InstanceName: sv.Instance(),
		TypeMap:      jm.typeMap,
		ObjMap:       objMap,
		StructMap:    structMap,
		Include:      jm.includeRegexpMap,
		Replace:      jm.replaceRegexpMap,
		ReplaceText:  jm.replaceTextMap,
		Duplicates:   make(map[*Parameters]map[string]*Duplicate),
	}, nil
}

// ReadSObjectDescriptions iterates through salesforce instance's objects and attaches them to
// the appropriate package.
func (cfg *Config) ReadSObjectDescriptions(ctx context.Context, sv *salesforce.Service) (*Job, error) {
	var el ErrorList
	var mErr sync.Mutex
	var checkError = func() bool {
		defer mErr.Unlock()
		mErr.Lock()
		return len(el) > 0
	}
	job, err := cfg.CreateJob(ctx, sv)
	if err != nil {
		return nil, err
	}
	var sendChannel = make(chan salesforce.SObjectDefinition)
	for i := 0; i < numberOfGoRoutines; i++ {
		job.wg.Add(1)
		go func() {
			for o := range sendChannel {
				if err := job.AssignSObjects(ctx, sv, o); err != nil {
					// TODO: adding better logging of errors for go routine
					log.Printf("unable to retreive info on %s, %v", o.Name, err)
					mErr.Lock()
					el = append(el, fmt.Errorf("unable to retreive info on %s, %v", o.Name, err))
					mErr.Unlock()
					break
				}
			}
			job.wg.Done()
		}()
	}
	for _, v := range job.ObjMap {
		if checkError() {
			break
		}
		sendChannel <- v
	}
	close(sendChannel)

	job.wg.Wait()
	if len(el) > 0 {
		return nil, el
	}
	return job, nil
}

func (job *Job) structOverride(cfg *Config, o *Override, p *Parameters, parent salesforce.SObjectDefinition) *Override {
	// check for a parent override.  If exists, use the parent override for naming
	parentOverride, ok := cfg.StructOverrides[parent.Name]
	if !ok || parentOverride.Name == "" {
		for idx := range cfg.Packages {
			// determine package for parent and create parent override for name
			px := &cfg.Packages[idx]
			if job.Match(px, &parent) {
				sx := job.Struct(px, &parent)
				var flds map[string]FldOverride

				if p.AssociatedIdentityType == "ChangeEvent" && parentOverride != nil && len(parentOverride.Fields) > 0 {
					flds = parentOverride.Fields
				} else if o != nil {
					flds = o.Fields
				}
				parentOverride = &Override{Name: sx.GoName, Fields: flds}
				break
			}
		}
	}
	return parentOverride
}

func (job *Job) structName(p *Parameters, objdef *salesforce.SObjectDefinition) *Override {
	cfg := job.Config
	goName := p.GoName(objdef)
	assocParentEntity := objdef.AssociateParentEntity

	// check for struct level override
	override, ok := cfg.StructOverrides[objdef.Name]
	if !ok || override.Name == "" {
		// if no override or blank name do replacement
		if replace := job.Replace[p]; replace != nil {
			if newName := replace.ReplaceAllString(goName, job.ReplaceText[p]); newName > "" {
				goName = newName
			}
		}

		// if associated type and type match between package and object, find parent
		// object because we will use the parent's name rather than the object's name
		if p.AssociatedIdentityType > "" && objdef.AssociateEntityType != nil &&
			p.AssociatedIdentityType == *objdef.AssociateEntityType {
			//assocParentEntity = *&objdef.AssociateParentEntity
			parent, ok := job.ObjMap[assocParentEntity]
			if ok {
				// check for a parent override.  If exists, use the parent override for naming
				override = job.structOverride(cfg, override, p, parent)
			}
		}
	}
	goName = override.GoName(goName)
	if override == nil {
		override = &Override{}
	}
	override.Name = goName
	return override
}

// Struct compile all needed data for outputting a go struct definition representing the sobject
func (job *Job) Struct(p *Parameters, objdef *salesforce.SObjectDefinition) *Struct {
	cfg := job.Config
	typeMap := sfTypeMap(job.TypeMap)
	override := job.structName(p, objdef)
	apiName := objdef.Name
	goName := override.Name
	var dupMap = make(map[string]int)
	var dupAPINameMap = make(map[string]string)
	var fields = make([]*Field, 0, len(objdef.Fields))

	for _, fld := range objdef.Fields {
		// selecdt basis for go field name
		goFieldName := fld.Name
		if p.UseLabel || (fld.Name[0] >= 97 && fld.Name[0] <= 122) {
			goFieldName = fld.Label
		}
		typeNm := typeMap.Get(fld.SoapType)
		skip := cfg.SkipRelationshipGlobal[fld.Name]
		goFld := override.Field(fld, goFieldName, typeNm, skip)

		// check for duplicate names in struct fields and append _DUP000 duplicate field
		oriGoName := goFld.GoName
		cnt, ok := dupMap[goFld.GoName]
		if ok { // previous field name exists add suffix to new

			job.addDuplicateField(p, apiName, DuplicateField{
				MatchingAPIName: dupAPINameMap[goFld.GoName],
				APIName:         goFld.APIName,
				Label:           fld.Label,
				GoName:          goFld.GoName,
			})
			log.Printf("Duplicate fields: type %s: %s %s", goName, fld.Name, goFld.GoName)
			goFld.GoName = goFld.GoName + fmt.Sprintf("_DUP%03d", cnt)
			cnt++
		} else {
			dupAPINameMap[goFld.GoName] = apiName
		}
		dupMap[oriGoName] = cnt
		fields = append(fields, goFld)
	}

	return &Struct{
		GoName:           goName,
		Label:            objdef.Label,
		APIName:          apiName,
		Receiver:         strings.ToLower(goName[0:1]),
		Readonly:         (!objdef.Updateable && !objdef.Createable),
		KeyPrefix:        objdef.KeyPrefix,
		AssociatedEntity: override.AssociateEntityName,
		FieldProps:       fields,
	}
}

// Job handles creation of package output
type Job struct {
	*Config
	InstanceName string
	TypeMap      map[string]string                       // map of SoapTypes to go types
	ObjMap       map[string]salesforce.SObjectDefinition //  map of all salesforce instance definitions
	StructMap    map[*Parameters][]Struct                // slice of Struct record by package config
	Include      map[*Parameters]*regexp.Regexp
	Replace      map[*Parameters]*regexp.Regexp
	ReplaceText  map[*Parameters]string
	Duplicates   map[*Parameters]map[string]*Duplicate
	wg           sync.WaitGroup
	m            sync.Mutex
}

func (job *Job) addDuplicate(p *Parameters, apiName string, dup Duplicate) {
	job.m.Lock()
	defer job.m.Unlock()
	if job.Duplicates == nil {
		job.Duplicates = make(map[*Parameters]map[string]*Duplicate)
	}
	if _, ok := job.Duplicates[p]; !ok {
		job.Duplicates[p] = make(map[string]*Duplicate)
	}

	d, ok := job.Duplicates[p][apiName]
	if ok {
		dup.Fields = d.Fields
	}
	job.Duplicates[p][apiName] = &dup
}

func (job *Job) addDuplicateField(p *Parameters, apiName string, dup DuplicateField) {
	job.m.Lock()
	defer job.m.Unlock()
	if job.Duplicates == nil {
		job.Duplicates = make(map[*Parameters]map[string]*Duplicate)
	}
	_, ok := job.Duplicates[p]
	if !ok {
		job.Duplicates[p] = make(map[string]*Duplicate, 0)
	}
	d, ok := job.Duplicates[p][apiName]
	if !ok {
		job.Duplicates[p][apiName] = &Duplicate{Fields: []DuplicateField{dup}}
		return
	}
	d.Fields = append(d.Fields, dup)
}

// Duplicate is an sobject whose goname previously exists in the definition.  Use
// to identify and create necessary overrides.
type Duplicate struct {
	MatchingAPIName string           `json:"matching_api_name,omitempty"`
	Label           string           `json:"label,omitempty"`
	GoName          string           `json:"go_name,omitempty"`
	Fields          []DuplicateField `json:"fields,omitempty"`
}

// DuplicateField provides parameter for a duplicated goname in a struct
type DuplicateField struct {
	MatchingAPIName string `json:"matching_api_name,omitempty"`
	APIName         string `json:"api_name,omitempty"`
	Label           string `json:"label,omitempty"`
	GoName          string `json:"go_name,omitempty"`
}

// AssignSObjects adds object definitions read from channel to appropriate packages
func (job *Job) AssignSObjects(ctx context.Context, sv *salesforce.Service,
	obj salesforce.SObjectDefinition) error {

	var p *Parameters
	cfg := job.Config
	for idx := range cfg.Packages {
		p = &cfg.Packages[idx]
		if job.Match(p, &obj) {
			// retreive full sobject fields
			objdef, err := sv.Describe(ctx, obj.Name)
			if err != nil {
				// TODO: adding better logging of errors for go routine
				log.Printf("unable to retreive info on %s, %v", obj.Name, err)
				return err
			}
			structDef := job.Struct(p, objdef)
			job.m.Lock()
			job.StructMap[p] = append(job.StructMap[p], *structDef)
			job.m.Unlock()
			break
		}
	}
	return nil
}

// TemplateData creates a simplified data structure for use with templates
func (job *Job) TemplateData(p *Parameters) *TemplateData {
	strx, ok := job.StructMap[p]
	if !ok {
		return nil
	}
	sort.Slice(strx, func(i, j int) bool {
		return strx[i].GoName < strx[j].GoName
	})
	if len(strx) > 1 {

		var prevName = strx[0].GoName
		var prevAPIName = strx[0].APIName
		var cnt int

		for idx := 1; idx < len(strx); idx++ {
			cur := strx[idx]
			if cur.GoName == prevName {
				cnt++
				strx[idx].GoName = fmt.Sprintf("%s_%03d", prevName, cnt)
				log.Printf("package %s duplicate goname %s with %s - %s", p.Name, prevAPIName, cur.APIName, cur.GoName)
				job.addDuplicate(p, cur.APIName, Duplicate{MatchingAPIName: prevAPIName, Label: cur.Label, GoName: cur.GoName})
				continue
			}
			prevName = cur.GoName
			prevAPIName = cur.APIName
			cnt = 0
		}
	}
	var duplicateJSON string
	if len(job.Duplicates[p]) > 0 {
		b, _ := json.MarshalIndent(job.Duplicates[p], "", "    ")
		duplicateJSON = string(b)
	}
	return &TemplateData{
		Name:                        p.Name,
		Description:                 strings.Replace(p.Description, "\n", "\n// ", -1),
		GoFilename:                  p.GoFilename,
		IncludeCodeGeneratedComment: job.Config.IncludeCodeGeneratedComment,
		Instance:                    job.InstanceName,
		Structs:                     strx,
		Duplicates:                  duplicateJSON,
	}
}

// Match checks whether an object definition matches the package file criteria
func (job *Job) Match(p *Parameters, obj *salesforce.SObjectDefinition) bool {
	// check for include listing as it overrides everything else
	if p.Include(obj.Name) {
		return true
	}
	if !(p.IncludeCustom && obj.Custom) && !(p.IncludeStandard && !obj.Custom) {
		return false
	}
	if p.AssociatedIdentityType > "" {
		if obj.AssociateEntityType == nil || p.AssociatedIdentityType != *obj.AssociateEntityType {
			return false
		}
	}

	include, ok := job.Include[p]
	if ok {
		return include.MatchString(obj.Name)
	}
	return true
}

// Field contains all fields for creating struct definition
type Field struct {
	GoName       string
	GoType       string
	Tag          string
	Comment      string
	APIName      string
	Relationship *Field
}

// TemplateData provides formatted data for a package's template exec
type TemplateData struct {
	Name                        string   `json:"name,omitempty"`
	Description                 string   `json:"description,omitempty"`
	GoFilename                  string   `json:"go_filename,omitempty"`
	IncludeCodeGeneratedComment bool     `json:"include_code_generated_comment,omitempty"`
	Instance                    string   `json:"instance,omitempty"`
	Structs                     []Struct `json:"structs,omitempty"`
	Duplicates                  string   `json:"duplicate_json"`
}

// Struct contains all needed information to create a salesforce.SObject
// definition
type Struct struct {
	GoName           string   `json:"name,omitempty"`
	Label            string   `json:"label,omitempty"`
	APIName          string   `json:"api_name,omitempty"`
	Receiver         string   `json:"receiver,omitempty"`
	Readonly         bool     `json:"readonly,omitempty"`
	KeyPrefix        string   `json:"keyPrefix,omitempty"`
	AssociatedEntity string   `json:"associated_entity,omitempty"`
	FieldProps       []*Field `json:"field_props,omitempty"`
}

// Parameters contains all data needed for generating a package
type Parameters struct {
	Description            string   `json:"description,omitempty"` // package documentation top line
	Name                   string   `json:"name,omitempty"`        // name of generated package
	GoFilename             string   `json:"go_filename,omitempty"`
	IncludeCustom          bool     `json:"include_custom,omitempty"`           // include custom objects
	IncludeStandard        bool     `json:"include_standard,omitempty"`         // include standard objecdts
	AssociatedIdentityType string   `json:"associated_identity_type,omitempty"` // include only associated types equal to value (use for Feed, Share, Change Event, etc.)
	IncludeNames           []string `json:"include,omitempty"`                  // list of objects to include in package
	IncludeMatch           string   `json:"include_match,omitempty"`            // include in package if Object Name matches any
	ReplaceMatch           string   `json:"replace_match,omitempty"`            // replace match in name
	ReplaceWith            string   `json:"replace_with,omitempty"`             // replace with this string if match
	UseLabel               bool     `json:"label_as_name,omitempty"`            // use Label field rather than name for calculating go name
}

// Include decides whether the sobject is in the IncludedNames list
func (p *Parameters) Include(nm string) bool {
	for _, s := range p.IncludeNames {
		if s == nm {
			return true
		}
	}
	return false
}

// GoName returns the field that will be the basis of the struct name
func (p *Parameters) GoName(objdef *salesforce.SObjectDefinition) string {
	goName := objdef.Name
	// check package setting and look for lowercase name to use label.  This catches all
	// npsp/np* namespace objects
	if p.UseLabel || (goName[0] >= 97 && goName[0] <= 122) {
		goName = objdef.Label
	} else {
		goName = strings.TrimSuffix(goName, "__c")
	}
	return goName
}

// Validate ensures that required fields contain appropriate data and
// attempts to compile any regexp statements
func (p *Parameters) Validate() (*regexp.Regexp, *regexp.Regexp, string, error) {
	var includeRegexp, replaceRegexp *regexp.Regexp
	var replacementText string
	var err error
	if p.Name == "" {
		return nil, nil, "", errors.New("package name not specified")
	}
	if p.IncludeMatch > "" {
		if includeRegexp, err = regexp.Compile(p.IncludeMatch); err != nil {
			return nil, nil, "", fmt.Errorf("package %s includematch regexp compile failed %s %w", p.Name, p.IncludeMatch, err)
		}
	}
	if p.ReplaceMatch > "" {
		if replaceRegexp, err = regexp.Compile(p.ReplaceMatch); err != nil {
			return nil, nil, "", fmt.Errorf("package %s replacematch regexp compile failed %s %w", p.Name, p.ReplaceMatch, err)
		}
		replacementText = p.ReplaceWith
	}
	if !p.IncludeCustom && !p.IncludeStandard && len(p.IncludeNames) == 0 && p.IncludeMatch == "" && p.AssociatedIdentityType == "" {
		return nil, nil, "", errors.New("no selection criteria specified")
	}

	return includeRegexp, replaceRegexp, replacementText, nil
}

// Override defines struct and fld overrides to set struct name.  The name
// is not tested against go linting rules.  Point will set the field as a ptr
// to allow zero values to be sent.
type Override struct {
	Name                string                 `json:"name,omitempty"`
	Fields              map[string]FldOverride `json:"fields,omitempty"`
	AssociateEntityName string                 `json:"associated_entity,omitempty"`
}

// GoName returns go name for struct
func (o *Override) GoName(nm string) string {
	if o != nil && o.Name > "" {
		return o.Name
	}
	return LintName(nm)
}

// FieldOverride returns all field overrides as well as linted go name
func (o *Override) FieldOverride(nm, lbl string) *FldOverride {
	if o == nil {
		return &FldOverride{Name: LintName(lbl)}
	}
	fo := o.Fields[nm]
	if fo.Name == "" {
		fo.Name = LintName(lbl)
	}
	return &fo
}

// FldOverride contains a replacement name and whether the field should be defined as a pointer
type FldOverride struct {
	Name             string `json:"name,omitempty"`
	IsPointer        bool   `json:"is_pointer,omitempty"`
	SkipRelationship bool   `json:"skip_relationship,omitempty"`
}

// Field determines comments, type, tag and name
func (o *Override) Field(fx salesforce.Field, goName string, typeNm string, skipRelationship bool) *Field {
	goName = strings.TrimSuffix(goName, "__c")
	override := o.FieldOverride(fx.Name, goName)
	fldNm := override.Name
	if override.SkipRelationship {
		skipRelationship = true
	}
	if override.IsPointer {
		typeNm = "*" + typeNm
	}

	proplbl := fieldPropertiesLabel(fx)
	ftype := fx.Type
	if fx.Length > 0 {
		ftype = fmt.Sprintf("%s(%d)", ftype, fx.Length)
	}
	fp := &Field{
		GoName:  fldNm,
		GoType:  typeNm,
		Tag:     fmt.Sprintf("`json:\"%s,omitempty\"`", fx.Name),
		APIName: fx.Name,
		Comment: strings.TrimLeft(proplbl+" "+ftype, " "),
	}
	// add relationship only if updateable
	if isAuditFieldRelationship(fx.Name) ||
		(!skipRelationship && len(fx.ReferenceTo) > 0 && (fx.Updateable || fx.Createable) && fx.RelationshipName > "") {
		fp.Relationship = &Field{
			GoName:  fldNm + "Rel",
			GoType:  "map[string]interface{}",
			Tag:     fmt.Sprintf("`json:\"%s,omitempty\"`", fx.RelationshipName),
			APIName: fx.RelationshipName,
			Comment: fmt.Sprintf("update with external id %v", fx.ReferenceTo),
		}
	}
	return fp
}

func fieldPropertiesLabel(fx salesforce.Field) string {
	var props []string
	if fx.ExternalID {
		props = append(props, "ExternalID")
	}
	if fx.AutoNumber {
		props = append(props, "AUTO-NUMBER")
	}
	if !fx.Updateable && !fx.Createable {
		props = append(props, "READ-ONLY")
		if fx.Calculated {
			props = append(props, "CALCULATED")
		}
	}
	if fx.HTMLFormatted {
		props = append(props, "HTML")
	}
	if fx.IDLookup {
		props = append(props, "LOOKUP")
	}

	//ftype := fx.Type
	//if fx.Length > 0 {
	//	ftype = fmt.Sprintf("%s(%d)", ftype, fx.Length)
	//}
	if len(props) > 0 {
		return "[" + strings.Join(props, " ") + "]"
	}
	return ""

}

func isAuditFieldRelationship(nm string) bool {
	return nm == "LastModifiedById" || nm == "CreatedById"
}

var alphanumOnly = regexp.MustCompile("_SYSTEM:|[^_a-zA-Z0-9]")

// LintName returns a properly linted go name based upon the orginal field name or label
func LintName(name string) (should string) {
	if len(name) == 0 {
		return "INVALID_blankname"
	}
	name = strings.Replace(name, " ", "_", -1)
	name = alphanumOnly.ReplaceAllString(name, "")
	for _, char := range name {
		if !(char >= 65 && char <= 90) && !(char >= 97 && char <= 122) {
			name = name[1:]
			if len(name) == 0 {
				return "INVALID_"
			}
			continue
		}
		break
	}
	name = strings.ToUpper(name[0:1]) + name[1:]

	return lint.Name(alphanumOnly.ReplaceAllString(name, ""), nil, nil)
}

type sfTypeMap map[string]string

func (tm sfTypeMap) Get(key string) string {
	s, ok := tm[key]
	if ok {
		return s
	}
	return "interface{}"
}

const defaultTemplateSource = `// Package {{.Name}} {{.Description}}{{if .IncludeCodeGeneratedComment}}
// Code generated for salesforce instance {{.Instance}}; DO NOT EDIT.{{else}}
// instance: {{.Instance}}{{end}}
package {{.Name}}

import (
	"github.com/jfcote87/salesforce"
)

{{range .Structs}}// {{.GoName}} describes the salesforce object {{.APIName}} {{.KeyPrefix}} ({{.Label}}){{if .Readonly}} [READ ONLY]{{end}}
type {{.GoName}} struct {
	Attributes *salesforce.Attributes ` + "`json:" + `"attributes,omitempty"` + "`" + ` 
{{range .FieldProps}}    {{.GoName}} {{.GoType}} {{.Tag}} // {{.Comment}}
{{if .Relationship}}    {{.Relationship.GoName}} {{.Relationship.GoType}} {{.Relationship.Tag}} // {{.Relationship.Comment}}
{{end}}{{end}}}

// SObjectName return rest api name of {{.APIName}}
func ({{.Receiver}} {{.GoName}}) SObjectName() string {
	return "{{.APIName}}"
}

// WithAttr returns a new {{.GoName}} with attributes of Type="{{.APIName}}"
// and Ref=ref
func({{.Receiver}} {{.GoName}}) WithAttr(ref string) salesforce.SObject {
	{{.Receiver}}.Attributes = &salesforce.Attributes{Type: "{{.APIName}}", Ref: ref }
	return {{.Receiver}}
}
{{end}}{{if .Duplicates}}
// Duplicate struct and field names
/* 
{{.Duplicates}}
*/{{end}}
`

// MakeSource creates formatted source code from Config parameters.  The returned map's keys are the go_filename from the
// PackageParams and the byte array is the generated and formatted code. If tmp is nil, the procedure uses the defaultTemplate.
func (cfg *Config) MakeSource(ctx context.Context, sv *salesforce.Service, tmpl *template.Template) (map[string][]byte, error) {
	tds, err := cfg.MakeTemplateData(ctx, sv)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		tmpl = defaultTemplate
	}
	fileMap := make(map[string][]byte)
	for _, td := range tds {
		if len(td.Structs) == 0 {
			continue
		}
		var tmplOut = &bytes.Buffer{}
		if err := tmpl.Execute(tmplOut, td); err != nil {
			return nil, err
		}
		fmtOut, err := format.Source(tmplOut.Bytes())
		if err != nil {
			return nil, err
		}
		fileMap[td.GoFilename] = fmtOut

	}
	return fileMap, nil
}
