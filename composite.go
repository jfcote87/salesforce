// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// RetrieveRecords returns sobjects rows pointed to by the passed ids, results must be a pointer
// to a slice of types implementing SObject interface.  If retrieving SObjects of different types,
// have the results be a *[]GenericSObject and read the objest' Attributes.Type field to identify
// the object type.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobjects_collections_retrieve.htm
func (sv *Service) RetrieveRecords(ctx context.Context, results interface{}, ids []string, fields ...string) error {
	var body = struct {
		IDS    []string `json:"ids"`
		Fields []string `json:"fields"`
	}{
		IDS:    ids,
		Fields: fields,
	}
	if len(ids) == 0 {
		return errors.New("no ids specified")
	}
	if len(fields) == 0 {
		return errors.New("no fields specified")
	}

	var ty = reflect.TypeOf((*SObject)(nil)).Elem()

	if results == nil {
		return errors.New("results parameter may not be nil")
	}

	resultsType := reflect.TypeOf(results)
	if resultsType.Kind() != reflect.Ptr ||
		resultsType.Elem().Kind() != reflect.Slice {
		return errors.New("results must be a pointer to a slice")
	}
	if !resultsType.Elem().Elem().Implements(ty) {
		return fmt.Errorf("%s is not an SObject", resultsType.Elem().Elem().Name())
	}

	err := sv.Call(ctx, fmt.Sprintf("composite/sobjects/%s", resultsType.Elem().Elem().Name()), "POST", body, results)
	return err
}

// GetRelatedRecords retrieves related records from an SObject's defined relationship.  result should be a pointer to
// a single SObject record when the relationship is one to one, otherwise use a pointer to a slice of a specific SObject.
// If no relationship exists in a one to one relationship, a 404 error is returned.  A one to many relationship will
// return an empty slice.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_sobject_relationships.htm
func (sv *Service) GetRelatedRecords(ctx context.Context, result interface{}, sobjectName, id, relationship string, fields ...string) error {
	if result == nil {
		return errors.New("results parameter may not be nil")
	}
	path := fmt.Sprintf("sobjects/%s/%s/%s", sobjectName, id, relationship)
	if len(fields) > 0 {
		path = path + "?fields=" + strings.Join(fields, ",")
	}
	return sv.Call(ctx, path, "GET", nil, result)
}

// DeleteRelatedRecord with detatch the record on the the defined relationship for one to one
// relationships.  Scroll the Example using DELETE to delete a relationship record section.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_relationship_traversal.htm#dome_relationship_traversal
func (sv *Service) DeleteRelatedRecord(ctx context.Context, sobjectName, id, relationship string) error {
	return sv.Call(ctx, fmt.Sprintf("sobjects/%s/%s/%s", sobjectName, id, relationship), "DELETE", nil, nil)
}

// UpdateRelatedRecord updates the record attached to the defined relationship.  Do not
// in the Id in the passed SObject. Scroll down to the Example of using PATCH to update
// a relationship record.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_relationship_traversal.htm#dome_relationship_traversal
func (sv *Service) UpdateRelatedRecord(ctx context.Context, updateRecord SObject, sobjectName, id, relationship string) error {
	return sv.Call(ctx, fmt.Sprintf("sobjects/%s/%s/%s", sobjectName, id, relationship), "PATCH", updateRecord, nil)
}

// BatchLogFunc is passed the corresponding SObjects and OpResponses created
// and returned from a salesforce batch.  The int value is the number of records
// previously processed.  Context is the context passed to the composite call. Returning
// a non-nil error halts the composite call.
type BatchLogFunc func(context.Context, int, []SObject, []OpResponse) error

// BatchBody is the body of a collection Create,Update,Upsert
type BatchBody struct {
	AllOrNone bool      `json:"allOrNone,omitempty"`
	Records   []SObject `json:"records,omitempty"`
}

// CreateRecords inserts records from recs.  Salesforce will return an error for any record that
// has a RecordID set.  The OpResponses will contain the new RecordIDs.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobjects_collections_create.htm
func (sv *Service) CreateRecords(ctx context.Context, allOrNone bool, recs []SObject) ([]OpResponse, error) {
	return sv.CompositeCall(ctx, allOrNone, "composite/sobjects", "POST", recs)
}

// UpdateRecords update records from recs.  The each record must set the Salesforce RecordID for the object.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobjects_collections_update.htm
func (sv *Service) UpdateRecords(ctx context.Context, allOrNone bool, recs []SObject) ([]OpResponse, error) {
	return sv.CompositeCall(ctx, allOrNone, "composite/sobjects", "PATCH", recs)
}

// UpsertRecords updates/inserts records based upon the external id field.  All recs must be of the same
// Object Type.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobjects_collections_upsert.htm
func (sv *Service) UpsertRecords(ctx context.Context, allOrNone bool, externalIDField string, recs []SObject) ([]OpResponse, error) {
	if len(recs) == 0 {
		return nil, ErrZeroRecords
	}
	sobjNm := recs[0].SObjectName()

	return sv.CompositeCall(ctx, allOrNone, fmt.Sprintf("composite/sobjects/%s/%s", sobjNm, externalIDField), "PATCH", recs)
}

// DeleteRecords deletes a list sobject from the list of ids
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobjects_collections_delete.htm
func (sv *Service) DeleteRecords(ctx context.Context, allOrNone bool, ids []string) ([]OpResponse, error) {
	if len(ids) <= 0 {
		return nil, ErrZeroRecords
	}
	var opResp = make([]OpResponse, 0, len(ids))
	batchSz := sv.MaxBatchSize()
	for i := 0; i < len(ids); i += batchSz {
		numRecs := i + batchSz
		if numRecs > len(ids) {
			numRecs = len(ids)
		}
		delIDs := ids[i:numRecs]
		path := "composite/sobjects?ids=" + strings.Join(delIDs, ",")
		var res []OpResponse
		if err := sv.Call(ctx, path, "DELETE", nil, &res); err != nil {
			return opResp, err
		}
		opResp = append(opResp, res...)
		var delrecids = make([]SObject, 0, len(res))
		for _, s := range delIDs {
			delrecids = append(delrecids, DeleteID(s))
		}
		if sv.logger != nil {
			if err := sv.logger(ctx, i, delrecids, res); err != nil {
				return nil, err
			}
		}
	}

	return opResp, nil
}

// CompositeCall updates/inserts/upserts all records in batches based upon the Service
// batch size (generally 200).
func (sv *Service) CompositeCall(ctx context.Context, allOrNone bool, path, method string, recs []SObject) ([]OpResponse, error) {
	if len(recs) == 0 {
		return nil, ErrZeroRecords
	}
	var opResp = make([]OpResponse, 0, len(recs))
	batchSz := sv.MaxBatchSize()

	for i := 0; i < len(recs); i += batchSz {
		cmdRecs := make([]SObject, 0, batchSz)
		numRecs := i + batchSz
		if numRecs > len(recs) {
			numRecs = len(recs)
		}
		for _, r := range recs[i:numRecs] {
			cmdRecs = append(cmdRecs, r.WithAttr(""))
		}
		body := BatchBody{AllOrNone: allOrNone, Records: cmdRecs}
		var res []OpResponse

		if err := sv.Call(ctx, path, method, body, &res); err != nil {
			return opResp, err
		}
		opResp = append(opResp, res...)
		if sv.logger != nil {
			if err := sv.logger(ctx, i, cmdRecs, res); err != nil {
				return opResp, err
			}
		}
	}
	return opResp, nil
}

// ErrZeroRecords indicates a zero length SObject slice is passed to collection func
var ErrZeroRecords = errors.New("must have at least 1 record")

// OpResponses is a slice of OpResponse records
type OpResponses []OpResponse

// Errors returns unsuccessful OpResponses
func (oprs OpResponses) Errors(startIndex int, sobjects []SObject) []OpResponse {
	var errReponses []OpResponse
	for i := range oprs {
		if !oprs[i].Success {
			if i < len(sobjects) {
				oprs[i].RecordIndex = i + startIndex
				oprs[i].SObject = sobjects[i]
			}
			errReponses = append(errReponses, oprs[i])
		}
	}
	return errReponses
}
