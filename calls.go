// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/jfcote87/ctxclient"
	"github.com/jfcote87/oauth2"
)

const currentAPIVersion = "v53.0"

const defaultContentType = "application/json; charset=UTF-8"
const defaultAccept = "application/json"

const defaultTokenDuration = time.Hour * 4

var sobjectType reflect.Type

func init() {
	var ty SObject
	sobjectType = reflect.TypeOf(ty)
}

// Service handles creation, authorization and execution of REST Api calls
// via its methods
type Service struct {
	baseURL     *url.URL
	cf          ctxclient.Func
	ts          oauth2.TokenSource
	isqry       bool
	batchSize   int
	maxrows     int
	contentType string
	accept      string
	logger      func(context.Context, int, []SObject, []OpResponse) error //BatchLogger
}

// New creates a salesforce service.  The host should be in the format
// <mydomain>.my.salesforce.com.  For a fuller explanation, see the URI section of
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest_resources.htm
// The cf parameter must return an *http.Client to that authorizes eacha call.
func New(host string, version string, ts oauth2.TokenSource) *Service {
	if version == "" {
		version = currentAPIVersion
	}
	baseURL, _ := url.Parse("https://" + host + "/services/data/" + version + "/")
	return &Service{baseURL: baseURL, ts: ts}
}

// WithCtxClientFunc returns a service that uses the ctxclient.Func for determining
// the http client to use for calls.  Use mainly for debugging and testing.
func (sv *Service) WithCtxClientFunc(f ctxclient.Func) *Service {
	svnew := *sv
	svnew.cf = f
	return &svnew
}

// WithAcceptContentType replaces default accept and contentType headers
// with passed values.  Use when needing to set/receive other than applicaton/json
// such text/csv or text/xml.  Empty strings in accept or contentType
// parameters assume existing values.
func (sv *Service) WithAcceptContentType(accept, contentType string) *Service {
	svnew := *sv
	svnew.contentType = contentType
	svnew.accept = accept
	return &svnew
}

// WithBatchSize returns a service that uses batchSz to regulate batches from query and
// collection update operations.  Queries use the setting to set the Sforce-Query-Options
// header which deteremines the number of records returned per batch.  The maximum number of
// returned rows is 2000 and minimum is 200. If batch size is set to less than 200, query operations
// will 200 as the batch size. Collection updates use the setting to determine
// the maximum number of update records per call, and the maximum setting is 200.  If the batch size
// setting is greater than 200, batch calls will use 200 as the setting.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/headers_queryoptions.htm?search_text=batchSize
func (sv *Service) WithBatchSize(batchSz int) *Service {
	snew := *sv
	if batchSz < 0 {
		batchSz = 0
	}
	snew.batchSize = batchSz

	return &snew
}

// WithURL creates a new service that uses the passed URL as the
// prefix for calls.  Created to allow testing with httptest
func (sv *Service) WithURL(newURL string) *Service {
	snew := *sv
	snew.baseURL, _ = url.Parse(newURL)
	return &snew
}

// WithMaxrows sets the max total rows returned for a query (not
// the rows in a batch for composite functions)
func (sv *Service) WithMaxrows(maxrows int) *Service {
	if maxrows < 0 {
		maxrows = 0
	}
	snew := *sv
	snew.maxrows = maxrows
	return &snew
}

// contentTypeHeader returns the service's content-type header
func (sv *Service) contentTypeHeader() string {
	if sv == nil || sv.contentType > "" {
		return sv.contentType
	}
	return defaultContentType
}

// acceptHeader returns the service's accept header
func (sv *Service) acceptHeader() string {
	if sv != nil && sv.accept > "" {
		return sv.accept
	}
	return defaultAccept
}

// MaxBatchSize returns the batchSize property and used to limit batch sizes
func (sv *Service) MaxBatchSize() int {
	var defaultMax, defaultMin = 2000, 200
	if !sv.isqry {
		defaultMax, defaultMin = 200, 1
	}
	if sv.batchSize == 0 || sv.batchSize > defaultMax {
		return defaultMax
	}
	if sv.batchSize < defaultMin {
		return defaultMin
	}
	return sv.batchSize
}

// WithLogger returns a new service that uses the passed BatchLogFunc
// in composite calls to review OpResponses after each batch allowing
// as processed logging instead of waiting for the end and
// reviewing every OpResponse
func (sv *Service) WithLogger(blf BatchLogFunc) *Service {
	if sv == nil {
		return sv
	}
	snew := *sv
	snew.logger = blf
	return &snew
}

// Instance returns the serviced instance
func (sv *Service) Instance() string {
	if sv == nil || sv.baseURL == nil {
		return ""
	}
	return sv.baseURL.Host
}

// HTTPBody allows salesforce calls to be returned as a stream rather than
// a decoded json object.
type HTTPBody struct {
	Rdr           io.ReadCloser
	ContentType   string
	ContentLength int64
}

func (sv *Service) generateRequest(ctx context.Context, method, path string,
	body io.Reader, setAccept bool) (*http.Request, error) {
	callURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse path %s; %v", path, err)
	}
	if !strings.HasPrefix(path, "/") {
		callURL.Path = sv.baseURL.Path + callURL.Path
	}
	callURL.Scheme = sv.baseURL.Scheme
	callURL.Host = sv.baseURL.Host

	// leave URL empty as callURL already constructed
	r, err := http.NewRequest(method, "", body)
	if err != nil {
		return nil, err
	}
	r.URL = callURL

	if sv.isqry {
		r.Header.Set("Sforce-Query-Options", fmt.Sprintf("batchSize=%d", sv.MaxBatchSize()))
	}
	if body != nil {
		r.Header.Set("Content-Type", sv.contentTypeHeader())
	}
	if setAccept {
		r.Header.Set("Accept", sv.acceptHeader())
	}
	if sv.ts != nil {
		tk, err := sv.ts.Token(ctx)
		if err != nil {
			return nil, err
		}
		tk.SetAuthHeader(r)
	}
	return r, nil
}

// Call performs all api operations.  All other service operations call
// this func, so rarely should there be a need to use directly.
//
// If path begins with "/", it will be used as
// an absolute path otherwise it is appended to the service's base path.
// body may be nil, io.Reader or an interface{}.  An interface{} is marshaled as json.
// result must be a pointer to an expected result type.
func (sv *Service) Call(ctx context.Context, path, method string, body interface{}, result interface{}) error {
	if sv == nil || sv.baseURL == nil {
		return errors.New("nil baseURL")
	}
	var rqBody io.Reader
	switch val := body.(type) {
	case nil:
		// skip nil
	case io.Reader:
		// set rqBody to reader
		rqBody = val
	default:
		// marshal body into byte reader
		b, _ := json.MarshalIndent(body, "", "    ")
		rqBody = bytes.NewReader(b)
	}
	r, err := sv.generateRequest(ctx, method, path, rqBody, result != nil)
	if err != nil {
		return err
	}

	res, err := sv.cf.Do(ctx, r)
	if err != nil {
		return err
	}
	switch rx := result.(type) {
	case **HTTPBody:
		if rx != nil {
			*rx = &HTTPBody{
				Rdr:           res.Body,
				ContentType:   res.Header.Get("Content-type"),
				ContentLength: res.ContentLength,
			}
			return nil
		}
		err = errors.New("result may not be a nil ptr")
	case interface{}: // non-nil value
		err = json.NewDecoder(res.Body).Decode(result)
	}
	res.Body.Close()
	return err
}

// ObjectList returns all objects with top level metadata
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_describeGlobal.htm
func (sv *Service) ObjectList(ctx context.Context) ([]SObjectDefinition, error) {
	var result = struct {
		Encoding     string              `json:"encoding,omitempty"`
		MaxBatchSize int                 `json:"maxBatchSize,omitempty"`
		Objects      []SObjectDefinition `json:"sobjects,omitempty"`
	}{}
	if err := sv.Call(ctx, "sobjects/", "GET", nil, &result); err != nil {
		return nil, err
	}
	return result.Objects, nil
}

// Describe returns all fields of an SObject along with top level metadata
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_sobject_describe.htm
func (sv *Service) Describe(ctx context.Context, name string) (*SObjectDefinition, error) {
	var result *SObjectDefinition
	err := sv.Call(ctx, fmt.Sprintf("sobjects/%s/describe", name), "GET", nil, &result)
	return result, err
}

// GetDeletedResponse contains a list of deleted ids
type GetDeletedResponse struct {
	DeletedRecords        []DeletedRecord `json:"deletedRecords,omitempty"`
	EarliestDateAvailable Datetime        `json:"earliestDateAvailable,omitempty"`
	LatestDateCovered     Datetime        `json:"latestDateCovered,omitempty"`
}

// DeletedRecord contains the id and delete date of a deleted record
type DeletedRecord struct {
	http.HandlerFunc
	ID          string   `json:"id,omitempty"`
	DeletedDate Datetime `json:"deletedDate,omitempty"`
}

// GetDeletedRecords returns a list of ids for records deleted in the time range
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_get_deleted.htm
func (sv *Service) GetDeletedRecords(ctx context.Context, sobjectName string, start, end time.Time) (*GetDeletedResponse, error) {
	var q = make(url.Values)
	q.Set("start", start.Format(time.RFC3339))
	q.Set("end", end.Format(time.RFC3339))
	path := fmt.Sprintf("sobjects/%s/deleted/?%s", sobjectName, q.Encode())
	var res *GetDeletedResponse
	return res, sv.Call(ctx, path, "GET", nil, &res)
}

// GetUpdatedResponse contains a list of updated ids
type GetUpdatedResponse struct {
	IDs               []string `json:"ids,omitempty"`
	LatestDateCovered Datetime `json:"latestDateCovered,omitempty"`
}

// GetUpdatedRecords returns a list of ids for records updated in the time range
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_get_updated.htm
func (sv *Service) GetUpdatedRecords(ctx context.Context, sobjectName string, start, end time.Time) (*GetUpdatedResponse, error) {
	var q = make(url.Values)
	q.Set("start", start.Format(time.RFC3339))
	q.Set("end", end.Format(time.RFC3339))
	path := fmt.Sprintf("sobjects/%s/updated/?%s", sobjectName, q.Encode())

	var res *GetUpdatedResponse
	return res, sv.Call(ctx, path, "GET", nil, &res)
}

// OpResponse is returned for each record of and Update, Upsert and Insert
type OpResponse struct {
	ID          string  `json:"id"`
	Success     bool    `json:"success"`
	Errors      []Error `json:"errors"`
	Created     bool    `json:"created,omitempty"`
	RecordIndex int     `json:"-"`
	SObject     SObject `json:"-"`
}

// SObjectValue attempts to assign the response SObject to the
// ix pointer.  See example.
//
// var cx Contact
// err := or.SObjectValue(&cx)
func (or OpResponse) SObjectValue(ix interface{}) error {
	soType := reflect.TypeOf(or.SObject)
	xval := reflect.ValueOf(ix)
	if xval.Kind() != reflect.Ptr {
		return fmt.Errorf("expected ptr received %s", xval.Kind().String())
	}
	xtype := xval.Type()
	if !soType.ConvertibleTo(xtype.Elem()) {
		return fmt.Errorf("%s not convertable to %s", soType.String(), xval.Type().String())
	}
	reflect.ValueOf(ix).Elem().Set(reflect.ValueOf(or.SObject).Convert(xtype.Elem()))
	return nil
}

// Create inserts a row
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_sobject_create.htm
func (sv *Service) Create(ctx context.Context, rec SObject) (*OpResponse, error) {
	var res *OpResponse
	return res, sv.Call(ctx, "sobjects/"+rec.SObjectName(), "POST", rec, &res)
}

// Update updates a row.  ID must not be set on the rec.
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_update_fields.htm
func (sv *Service) Update(ctx context.Context, rec SObject, id string) error {
	return sv.Call(ctx, "sobjects/"+rec.SObjectName()+"/"+id, "PATCH", rec, nil)
}

// Delete deletes a row
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_delete_record.htm
func (sv *Service) Delete(ctx context.Context, sobjectName string, id string) error {
	return sv.Call(ctx, "sobjects/"+sobjectName+"/"+id, "DELETE", nil, nil)
}

// Get retrieves values of a single record identified by sf ID. The result parameterf
// must be a pointer to an SObject
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_get_field_values.htm
func (sv *Service) Get(ctx context.Context, result interface{}, id string, flds ...string) error {
	sobj, err := isSObjectPointer(result)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("sobjects/%s/%s?fields=%s", sobj.SObjectName(), id, strings.Join(flds, ","))
	return sv.Call(ctx, path, "GET", nil, result)
}

// GetByExternalID retrieves values of a single record identified by external ID. The result parameter
// must be a pointer to an SObject
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/using_resources_retrieve_with_externalid.htm
func (sv *Service) GetByExternalID(ctx context.Context, result interface{}, externalIDField, externalID string, flds ...string) error {
	sobj, err := isSObjectPointer(result)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("sobjects/%s/%s/%s?fields=%s", sobj.SObjectName(), externalIDField, externalID, strings.Join(flds, ","))
	return sv.Call(ctx, path, "GET", nil, result)
}

func isSObjectPointer(result interface{}) (SObject, error) {
	resType := reflect.TypeOf(result)
	resVal := reflect.ValueOf(result)
	if resType.Kind() != reflect.Ptr || resVal.IsNil() || !resVal.Elem().CanInterface() {
		return nil, fmt.Errorf("expected result to be a non-nil pointer; got %s", resType.Name())
	}
	ptx := resVal.Elem()
	sobj, ok := ptx.Interface().(SObject)
	if ok && ptx.Kind() == reflect.Ptr && ptx.IsNil() {
		sobj, ok = reflect.New(ptx.Type().Elem()).Interface().(SObject)
	}
	if !ok {
		return nil, errors.New("unable to convert result ptr to an SObject")
	}
	return sobj, nil

}

// Upsert inserts/updates a row using an external id
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_upsert.htm
func (sv *Service) Upsert(ctx context.Context, rec SObject, externalIDField, externalID string) (*OpResponse, error) {
	var res *OpResponse
	path := "sobjects/" + rec.SObjectName() + "/" + externalIDField + "/" + externalID
	return res, sv.Call(ctx, path, "PATCH", rec, &res)
}

// Query executes the query. All results are decoded into the results parameter that
// must be of the form *[]<struct>.  To set the f
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_query.htm
func (sv *Service) Query(ctx context.Context, qry string, results interface{}) error {
	return sv.query(ctx, "query/?q=", qry, results)
}

// QueryAll executes the query that will include filtering on deleted records
// https://developer.salesforce.com/docs/atlas.en-us.232.0.api_rest.meta/api_rest/dome_queryall.htm
func (sv *Service) QueryAll(ctx context.Context, qry string, results interface{}) error {
	return sv.query(ctx, "queryAll/?q=", qry, results)
}

func (sv *Service) query(ctx context.Context, path, qry string, results interface{}) error {
	switch results.(type) {
	case nil:
		return errors.New("results parameter may not be nil")
	}

	rs, err := NewRecordSlice(results)
	if err != nil {
		return err
	}
	var res = &QueryResponse{
		Records: rs,
	}
	fmtQry := path + url.QueryEscape(qry)
	qsv := *sv
	qsv.isqry = true
	for !res.Done {
		err = qsv.Call(ctx, fmtQry, "GET", nil, res)
		if err != nil {
			return err
		}
		if sv.maxrows > 0 {
			if rs.rows() >= sv.maxrows {
				rs.slice(0, sv.maxrows)
				return nil
			}
		}
		fmtQry = res.NextRecordsURL
	}
	return nil

}

// TODO: create/update binary
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_sobject_insert_update_blob.htm

// GetAttachment retrieves a binary file from an attachment sobject
// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_sobject_blob_retrieve.htm
func (sv *Service) GetAttachment(ctx context.Context, sobjectName, id string) (*HTTPBody, error) {
	var rdr *HTTPBody
	if err := sv.WithAcceptContentType("*/*", "").Call(ctx, "sobjects/"+sobjectName+"/"+id, "GET", nil, &rdr); err != nil {
		return nil, err
	}
	return rdr, nil
}

// CreateJob is the beginning of a batch job
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/create_job.htm
func (sv *Service) CreateJob(ctx context.Context, jd *JobDefinition) (*Job, error) {
	var result *Job
	err := sv.Call(ctx, "jobs/ingest/", "POST", jd, &result)
	return result, err
}

// UploadJobDataFile sends csv file to SF
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/upload_job_data.htm
func (sv *Service) UploadJobDataFile(ctx context.Context, job string, fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	return sv.UploadJobData(ctx, job, f)
}

// UploadJobData sends csv stream to SF.  If rdr is an io.Closer, function will close stream.
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/upload_job_data.htm
func (sv *Service) UploadJobData(ctx context.Context, job string, rdr io.Reader) error {
	if rdrc, ok := rdr.(io.Closer); ok {
		defer rdrc.Close()
	}
	path := fmt.Sprintf("jobs/ingest/%s/batches", job)
	return sv.WithAcceptContentType("application/json", "text/csv").Call(ctx, path, "PUT", rdr, nil)
}

// CloseJob starts job processing
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/close_job.htm
func (sv *Service) CloseJob(ctx context.Context, jobID string) (*Job, error) {
	var result *Job
	var mx = map[string]string{"state": "UploadComplete"}
	err := sv.Call(ctx, "jobs/ingest/"+jobID, "PATCH", mx, &result)
	return result, err
}

// AbortJob stops job processing
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/close_job.htm
func (sv *Service) AbortJob(ctx context.Context, jobID string) (*Job, error) {
	var result *Job
	var mx = map[string]string{"state": "Aborted"}
	err := sv.Call(ctx, "jobs/ingest/"+jobID, "PATCH", mx, &result)
	return result, err
}

// DeleteJob ends a job
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/delete_job.htm
func (sv *Service) DeleteJob(ctx context.Context, jobID string) error {

	err := sv.Call(ctx, "jobs/ingest/"+jobID, "DELETE", nil, nil)
	return err
}

// GetJob returns status
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/get_job_info.htm
func (sv *Service) GetJob(ctx context.Context, jobID string) (*Job, error) {
	var result *Job
	err := sv.Call(ctx, "jobs/ingest/"+jobID, "GET", nil, &result)
	return result, err
}

// GetSuccessfulJobRecords returns recs
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/get_job_successful_results.htm
func (sv *Service) GetSuccessfulJobRecords(ctx context.Context, jobID string) (*HTTPBody, error) {
	path := fmt.Sprintf("jobs/ingest/%s/successfulResults/", jobID)
	var sr *HTTPBody
	sv2 := sv.WithAcceptContentType("text/csv", "")
	err := sv2.Call(ctx, path, "GET", nil, &sr)
	if err != nil {
		return nil, err
	}
	return sr, err
}

// GetFailedJobRecords returns recs
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/get_job_failed_results.htm
func (sv *Service) GetFailedJobRecords(ctx context.Context, jobID string) (*HTTPBody, error) {
	path := fmt.Sprintf("jobs/ingest/%s/failedResults/", jobID)
	var sr *HTTPBody
	err := sv.WithAcceptContentType("text/csv", "").Call(ctx, path, "GET", nil, &sr)
	if err != nil {
		return nil, err
	}
	return sr, err
}

// GetUnprocessedJobRecords returns recs
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/get_job_unprocessed_results.htm
func (sv *Service) GetUnprocessedJobRecords(ctx context.Context, jobID string) (*HTTPBody, error) {
	path := fmt.Sprintf("jobs/ingest/%s/unprocessedrecords/", jobID)
	var sr *HTTPBody
	err := sv.WithAcceptContentType("text/csv", "").Call(ctx, path, "GET", nil, &sr)
	if err != nil {
		return nil, err
	}
	return sr, err
}

// ListJobs returns all jobs status.  nextURL should be empty on first call
// and use the nextRecordsURL from the returned JobList
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/get_all_jobs.htm
func (sv *Service) ListJobs(ctx context.Context, nextURL string) (*JobList, error) {
	var result *JobList
	if nextURL > "" {
		return result, sv.WithURL(nextURL).Call(ctx, "", "GET", nil, &result)
	}
	return result, sv.Call(ctx, "jobs/ingest/", "GET", nil, &result)
}

// BulkQuery is passed to QueryCreateJob.  Query is
// processed by the job
type BulkQuery struct {
	Query string `json:"query,omitempty"`
	// Valid delimiters
	// COMMA (,) default
	// BACKQUOTE—backquote character (`)
	// CARET—caret character (^)
	// COMMA—comma character (,)
	// PIPE—pipe character (|)
	// SEMICOLON—semicolon character (;)
	// TAB—tab character
	ColumnDelimiter string `json:"columnDelimiter,omitempty"` // COMMA default
	// Valid line breaks
	// LF default
	// CRLF carriage return character followed by a linefeed character
	LineEnding string `json:"lineEnding,omitempty"`
}

// QueryCreateJob runs a query in a job. To include deleted records, set
// queryAll to true.
// https://developer.salesforce.com/docs/atlas.en-us.api_bulk_v2.meta/api_bulk_v2/query_create_job.htm
func (sv *Service) QueryCreateJob(ctx context.Context, bulkQuery BulkQuery, queryAll bool) (*Job, error) {
	op := "query"
	if queryAll {
		op = "queryAll"
	}
	var body = struct {
		Operation   string `json:"operation,omitempty"`
		ContentType string `json:"contentType,omitempty"`
		BulkQuery
	}{
		Operation:   op,
		ContentType: "CSV",
		BulkQuery:   bulkQuery,
	}
	var jobInfo *Job
	return jobInfo, sv.Call(ctx, "jobs/query", "POST", body, &jobInfo)
}

// DeleteID allows a string to be used as an SObject
type DeleteID string

// SObjectName simply return definition name
func (d DeleteID) SObjectName() string {
	return "DeleteID"
}

// WithAttr just returns object as it is not a struct
func (d DeleteID) WithAttr(ref string) SObject {
	return d
}
