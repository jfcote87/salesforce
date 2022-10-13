// Copyright 2022 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package salesforce_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jfcote87/ctxclient"
	"github.com/jfcote87/oauth2"
	"github.com/jfcote87/salesforce"
)

func TestService_Query(t *testing.T) {
	svx := salesforce.Service{}
	_ = svx
}

func getTest(fn string, results interface{}) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(results)
}

func testQueryHTTPServer(testAccessToken string) (*httptest.Server, error) {
	var rows []Contact
	if err := getTest("testfiles/get/qryrows.json", &rows); err != nil {
		return nil, err
	}
	var testfunc http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "expected GET", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+testAccessToken {
			http.Error(w, fmt.Sprintf("expected authorization header Bearer %s; got %s", testAccessToken, r.Header.Get("Authorization")), http.StatusUnauthorized)
			return
		}
		qry := r.URL.Query().Get("q")
		if !(r.URL.Path == "/query/" || r.URL.Path == "/query/nextset/") || qry == "" {
			http.Error(w, fmt.Sprintf("expected /query/?q=...; got %s?%s", r.URL.Path, r.URL.Query().Encode()), http.StatusBadRequest)
			return
		}
		rx := &salesforce.QueryResponse{}
		var totRows = 660
		var batch int = 200
		_, _ = fmt.Sscanf(r.Header.Get("Sforce-Query-Options"), "batchSize=%d", &batch)
		start, _ := strconv.Atoi(r.URL.Query().Get("s"))
		switch qry {
		case "firstset":
			outputRows := rows[:batch]
			rx.NextRecordsURL = fmt.Sprintf("query/nextset/?q=secondset&s=%d", batch)
			rx.TotalSize = totRows
			rx.Records, _ = salesforce.NewRecordSlice(&outputRows)
		case "secondset":
			if start == 0 {
				http.Error(w, "expected start value > 0 and < 32 ", http.StatusBadRequest)
				return
			}
			end := start + batch
			if end > totRows {
				end = totRows
			}
			outputRows := rows[start:end]
			rx.NextRecordsURL = fmt.Sprintf("/query/nextset/?q=secondset&s=%d", end)
			if end >= totRows {
				rx.Done = true
				rx.NextRecordsURL = ""
			}
			rx.TotalSize = totRows
			rx.Records, _ = salesforce.NewRecordSlice(&outputRows)

		default:
			http.Error(w, fmt.Sprintf("invalid path: %s", r.URL.Path), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(*rx)
	}
	return httptest.NewServer(testfunc), nil
}

func TestQueryCall(t *testing.T) {
	var testAccessToken = "ABCDEFGHIJKLMN"

	ts, err := testQueryHTTPServer(testAccessToken)
	if err != nil {
		t.Errorf("TestQueryCall http server start failed; %v", err)
		return
	}
	serverURL := ts.URL + "/"
	defer ts.Close()

	tk := &oauth2.Token{AccessToken: testAccessToken}
	sv := salesforce.New("aninstance.my.salesforce", "", oauth2.StaticTokenSource(tk)).
		WithURL(serverURL).WithBatchSize(5)

	ctx := context.Background()

	if err := sv.Query(ctx, "abc", nil); err == nil || err.Error() != "results parameter may not be nil" {
		t.Errorf("expected results parameter may not be nil; got %v", err)
	}

	if err := sv.Query(ctx, "abc", []string{""}); err == nil || !strings.HasPrefix(err.Error(), "expected *[]<struct>") {
		t.Errorf("expected results parameter may not be nil; got %v", err)
	}

	var resultRows []Contact
	if err = sv.Query(ctx, "firstset", &resultRows); err != nil {
		t.Errorf("firstset: %v", err)
		return
	}
	if len(resultRows) != 660 {
		t.Errorf("full read expected 660 rows; got %d", len(resultRows))
		return
	}
	var newRows []Contact
	var rowsLimit = 8
	if err = sv.WithMaxrows(rowsLimit).Query(ctx, "firstset", &newRows); err != nil {
		t.Errorf("resultset read: %v", err)
		return
	}
	if len(newRows) != rowsLimit {
		t.Errorf("full read expected %d rows; got %d", rowsLimit, len(resultRows))
		return
	}

	err = sv.WithMaxrows(rowsLimit).Query(ctx, "firstsetx", &newRows)
	notSuccess, ok := err.(*ctxclient.NotSuccess)
	if !ok || notSuccess.StatusCode != 404 {
		t.Errorf("expected 404 error; received %v", err)
		return
	}

}

type callTests struct {
	host                  string
	sv                    *salesforce.Service
	ctxOK, ctx400, ctx401 context.Context
}

func (ct callTests) testService_ObjectList(t *testing.T) {
	objs, err := ct.sv.ObjectList(ct.ctx401)
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 401 {
		t.Errorf("ctx401 expected 401 error; got %v", err)
		return
	}

	objs, err = ct.sv.ObjectList(context.Background())
	if err == nil || !strings.HasSuffix(err.Error(), "empty token") {
		t.Errorf("expected empty token error; got %v", err)
		return
	}
	objs, err = ct.sv.ObjectList(ct.ctxOK)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if len(objs) != 810 {
		t.Errorf("expected 810 records; got %d", len(objs))
	}
}

func (ct callTests) testService_Describe(t *testing.T) {
	desc, err := ct.sv.Describe(ct.ctx401, "Contact")
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 401 {
		t.Errorf("ctx401 expected 401 error; got %v", err)
		return
	}
	desc, err = ct.sv.Describe(ct.ctxOK, "Contact")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(desc.Fields) != 62 {
		t.Errorf("expected Contact to have 62 fields; got %d", len(desc.Fields))
	}

}

func (ct callTests) testService_GetDeleted(t *testing.T) {
	start, end := time.Now().Add(30*24*time.Hour), time.Now()
	dels, err := ct.sv.GetDeletedRecords(ct.ctx401, "Contact", start, end)
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 401 {
		t.Errorf("ctx401 expected 401 error; got %v", err)
		return
	}
	dels, err = ct.sv.GetDeletedRecords(ct.ctxOK, "Contact", start, end)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(dels.DeletedRecords) != 1 {
		t.Errorf("expected 1 delete record; go %d", len(dels.DeletedRecords))
	}
}

func (ct callTests) testService_GetUpdated(t *testing.T) {
	start, end := time.Now().Add(30*24*time.Hour), time.Now()
	upd, err := ct.sv.GetUpdatedRecords(ct.ctx401, "Contact", start, end)
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 401 {
		t.Errorf("ctx401 expected 401 error; got %v", err)
		return
	}
	upd, err = ct.sv.GetUpdatedRecords(ct.ctxOK, "Contact", start, end)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(upd.IDs) != 7 {
		t.Errorf("expected 7 records; go %d", len(upd.IDs))
	}
}

func (ct callTests) testService_Create(t *testing.T) {
	acctRecord := &Account{
		AccountName: "My Account",
		AccountType: "Business",
		VendorID:    "0123456",
	}
	opresp, err := ct.sv.Create(ct.ctx400, acctRecord)
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 400 {
		t.Errorf("ctx401 expected 400 error; got %v", err)
		return
	}
	opresp, err = ct.sv.Create(ct.ctxOK, acctRecord)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if !(opresp.Created && opresp.ID == "NEWID") {
		t.Errorf("expected created flag == true and ID == NEWID; got %v, %s", opresp.Created, opresp.ID)
	}
}

func (ct callTests) testService_Update(t *testing.T) {
	acctRecord := &Account{
		AccountName: "My Account",
		AccountType: "Business",
		VendorID:    "0123456",
	}
	id := "a1f4S000000cj9mQAA"
	err := ct.sv.Update(ct.ctx400, acctRecord, id)
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 400 {
		t.Errorf("ctx401 expected 400 error; got %v", err)
		return
	}
	err = ct.sv.Update(ct.ctxOK, acctRecord, id)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}

}

func (ct callTests) testService_Upsert(t *testing.T) {
	acctRecord := &Account{
		AccountName: "My Account",
		AccountType: "Business",
		VendorID:    "VN12345",
	}

	opresp, err := ct.sv.Upsert(ct.ctx400, acctRecord, "Vendor_ID__c", "VN12345")
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 400 {
		t.Errorf("ctx401 expected 400 error; got %v", err)
		return
	}
	opresp, err = ct.sv.Upsert(ct.ctxOK, acctRecord, "Vendor_ID__c", acctRecord.VendorID)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if !(!opresp.Created && opresp.ID == "") {
		t.Errorf("expected created flag == false and ID == \"\"; got %v, %s", opresp.Created, opresp.ID)
	}
	acctRecord.VendorID = "VN12346"
	opresp, err = ct.sv.Upsert(ct.ctxOK, acctRecord, "Vendor_ID__c", acctRecord.VendorID)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if !(opresp.Created && opresp.ID == "NEWID") {
		t.Errorf("expected created flag == true and ID == NEWID; got %v, %s", opresp.Created, opresp.ID)
	}
}

func (ct callTests) testService_Delete(t *testing.T) {
	delID := "a1f4S000000cj9mQAA"
	if err := ct.sv.Delete(ct.ctxOK, "Account", delID); err != nil {
		t.Errorf("expected success; got %v", err)
	}
}

func (ct callTests) testService_Get(t *testing.T) {
	var errstr00 = "unable to convert result ptr to an SObject"
	var errstr01 = "expected result to be a non-nil pointer"
	id := "a1f4S000000cj9mQAA"
	flds := []string{"Name", "Type", "RecordType", "Vendor_ID__C"}
	var resultA = 5
	if err := ct.sv.Get(ct.ctxOK, &resultA, id, flds...); err == nil ||
		!strings.HasSuffix(err.Error(), errstr00) {
		t.Errorf("expected %s; got %v", errstr00, err)
	}
	if err := ct.sv.Get(ct.ctxOK, resultA, id, flds...); err == nil ||
		!strings.HasPrefix(err.Error(), errstr01) {
		t.Errorf("wanted %s; %v", errstr01, err)
	}
	var resultB *Account
	if err := ct.sv.Get(ct.ctxOK, &resultB, id, flds...); err != nil {
		t.Errorf("expected success; got %v", err)
	}
}

func (ct callTests) testService_GetByExternalID(t *testing.T) {
	var errstr00 = "unable to convert result ptr to an SObject"
	id := "VN12345"
	flds := []string{"Name", "Type", "RecordType", "Vendor_ID__C"}
	var resultA = 5
	if err := ct.sv.GetByExternalID(ct.ctxOK, &resultA, "Vendor_ID__c", id, flds...); err == nil ||
		!strings.HasSuffix(err.Error(), errstr00) {
		t.Errorf("expected %s; got %v", errstr00, err)
	}

	var resultB *Account
	if err := ct.sv.GetByExternalID(ct.ctxOK, &resultB, "Vendor_ID__c", id, flds...); err != nil {
		t.Errorf("expected success; got %v", err)
	}
}

func (ct callTests) testService_GetAttachment(t *testing.T) {
	body, err := ct.sv.GetAttachment(ct.ctx401, "Attachment", "att4S000000cj9mQAA")
	ex, ok := err.(*ctxclient.NotSuccess)
	if !ok || ex.StatusCode != 401 {
		t.Errorf("ctx401 expected 401 error; got %v", err)
		return
	}
	body, err = ct.sv.GetAttachment(ct.ctxOK, "Attachment", "att4S000000cj9mQAA")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	defer body.Rdr.Close()
	buff := &bytes.Buffer{}
	numBytes, err := io.Copy(buff, body.Rdr)
	if err != nil {
		t.Errorf("expected read success; got %v", err)
		return
	}
	if numBytes != 8316 {
		t.Errorf("expected 8316 bytes; got %d", numBytes)
	}

}

func (ct callTests) testService_CreateJob(t *testing.T) {
	jc := &salesforce.JobDefinition{
		Object:              "Account",
		Operation:           "upsert",
		ExternalIDFieldName: "Vendor_ID__c",
	}
	ji, err := ct.sv.CreateJob(ct.ctxOK, jc)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if ji.ID != "JOB0000" {
		t.Errorf("expected jobinfo.ID = JOB0000; got %s", ji.ID)
	}
}

func (ct callTests) testService_UploadJobData(t *testing.T) {
	if err := ct.sv.UploadJobDataFile(ct.ctxOK, "JOB0000", "testfiles/pub/acctnot.csv"); err == nil || !strings.HasSuffix(err.Error(), "no such file or directory") {
		t.Errorf("expected no such file or directory; got %v", err)
	}

	if err := ct.sv.UploadJobDataFile(ct.ctxOK, "JOB0000", "testfiles/put/acct.csv"); err != nil {
		t.Errorf("expected success; got %v", err)
	}
}

func (ct callTests) testService_CloseJob(t *testing.T) {
	ji, err := ct.sv.CloseJob(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if ji.State != "UploadComplete" {
		t.Errorf("expected state of UploadComplete; got %s", ji.State)
	}
}

func (ct callTests) testService_AbortJob(t *testing.T) {
	ji, err := ct.sv.AbortJob(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if ji.State != "Aborted" {
		t.Errorf("expected state of Aborted; got %s", ji.State)
	}
}

func (ct callTests) testService_DeleteJob(t *testing.T) {
	err := ct.sv.DeleteJob(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
}

func (ct callTests) testService_GetJob(t *testing.T) {
	ji, err := ct.sv.GetJob(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if ji.State != "JobComplete" {
		t.Errorf("expected state of JobComplete; got %s", ji.State)
	}
}

func (ct callTests) testService_GetSuccessfulJobRecords(t *testing.T) {
	body, err := ct.sv.GetSuccessfulJobRecords(ct.ctxOK, "JOB000A")
	notSuccess, ok := err.(*ctxclient.NotSuccess)
	if !ok || notSuccess.StatusCode != 404 {
		t.Errorf("expected 404 error; received %v", err)
		return
	}

	body, err = ct.sv.GetSuccessfulJobRecords(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	defer body.Rdr.Close()
	wc := csv.NewReader(body.Rdr)
	rows, err := wc.ReadAll()
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows; got %d", len(rows))
	}
}

func (ct callTests) testService_GetFailedJobRecords(t *testing.T) {
	body, err := ct.sv.GetFailedJobRecords(ct.ctxOK, "JOB000A")
	notSuccess, ok := err.(*ctxclient.NotSuccess)
	if !ok || notSuccess.StatusCode != 404 {
		t.Errorf("expected 404 error; received %v", err)
		return
	}

	body, err = ct.sv.GetFailedJobRecords(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	defer body.Rdr.Close()
	wc := csv.NewReader(body.Rdr)
	rows, err := wc.ReadAll()
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows; got %d", len(rows))
	}
}

func (ct callTests) testService_GetUnprocessedJobRecords(t *testing.T) {
	body, err := ct.sv.GetUnprocessedJobRecords(ct.ctxOK, "JOB000A")
	notSuccess, ok := err.(*ctxclient.NotSuccess)
	if !ok || notSuccess.StatusCode != 404 {
		t.Errorf("expected 404 error; received %v", err)
		return
	}

	body, err = ct.sv.GetUnprocessedJobRecords(ct.ctxOK, "JOB0000")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	defer body.Rdr.Close()
	wc := csv.NewReader(body.Rdr)
	rows, err := wc.ReadAll()
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows; got %d", len(rows))
	}
}

func (ct callTests) testService_ListJobs(t *testing.T) {
	joblist, err := ct.sv.ListJobs(ct.ctxOK, "")
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if joblist.NextRecordsURL == "" {
		t.Errorf("expected NextRecordsURL")
		return
	}
	_, err = ct.sv.ListJobs(ct.ctxOK, ct.host+joblist.NextRecordsURL)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
}

func (ct callTests) testService_QueryCreateJob(t *testing.T) {
	bq := salesforce.BulkQuery{
		Query:           "Select ID FROM Account",
		ColumnDelimiter: "TAB",
	}
	ji, err := ct.sv.QueryCreateJob(ct.ctxOK, bq, true)
	if err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	if ji.Operation != "queryAll" {
		t.Errorf("expected operation queryAll; got %s", ji.Operation)
	}
}

func (ct callTests) testService_RetrieveRecords(t *testing.T) {
	var ids = []string{"0033000002239QCA", "003300000223aQCA", "003300000223bQCA", "003300000223cQCA"}
	var flds = []string{"Id", "AccountId", "FirstName", "External_ID__c"}
	var recs []Contact
	var nilResults = "results parameter may not be nil"
	var notptrslice = "results must be a pointer to a slice"
	var contactPtr *Contact
	var nrec []NotSObject
	var notSObject = "NotSObject is not an SObject"
	var tests = []struct {
		name   string
		cx     context.Context
		result interface{}
		ids    []string
		flds   []string
		errMsg string
	}{
		{name: "t00", cx: ct.ctxOK, result: &recs, flds: flds, errMsg: "no ids specified"},
		{name: "t01", cx: ct.ctxOK, result: &recs, ids: ids, errMsg: "no fields specified"},
		{name: "t02", cx: ct.ctxOK, ids: ids, flds: flds, errMsg: nilResults},
		{name: "t03", cx: ct.ctxOK, result: contactPtr, ids: ids, flds: flds, errMsg: notptrslice},
		{name: "t04", cx: ct.ctxOK, result: &nrec, ids: ids, flds: flds, errMsg: notSObject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ct.sv.RetrieveRecords(tt.cx, tt.result, tt.ids, tt.flds...)
			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("expected %s; got %v", tt.errMsg, err)
			}
		})
	}

	if err := ct.sv.RetrieveRecords(ct.ctxOK, &recs, ids, flds...); err != nil {
		t.Errorf("expected success; got %v", err)
		return
	}
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	for _, record := range recs {
		if !idMap[record.ContactID] {
			t.Errorf("%s not expected", record.ContactID)
		}
	}
}

func (ct *callTests) testService_Call(t *testing.T) {
	var sv *salesforce.Service
	if err := sv.Call(ct.ctxOK, "/abc", "qqqq", nil, nil); err == nil || err.Error() != "nil baseURL" {
		t.Errorf("expected nil baseURL; got %v", err)
	}
	sv = ct.sv
	if err := sv.Call(ct.ctxOK, "/abc", " _", nil, nil); err == nil || !strings.HasPrefix(err.Error(), "net/http: invalid method") {
		t.Errorf("expected net/http: invalid method; got %v", err)
	}
	if err := sv.Call(ct.ctxOK, "%!2@/abc", " ", nil, nil); err == nil || !strings.HasPrefix(err.Error(), "unable to parse path") {
		t.Errorf("unable to parse path; got %v", err)
	}
	var thttpbody **salesforce.HTTPBody
	if err := sv.Call(ct.ctxOK, "/sobjects/Attachment/att4S000000cj9mQAA", "GET", nil, thttpbody); err == nil || err.Error() != "result may not be a nil ptr" {
		t.Errorf("unable to parse path; got %v", err)
	}

}

func TestService_Call(t *testing.T) {
	ws := httptest.NewServer(http.HandlerFunc(serviceHandlerFunc))
	defer ws.Close()

	ct := &callTests{
		host: ws.URL,
		sv: salesforce.New("aninstance.my.salesforce", "", nil).WithCtxClientFunc(getTokenClientFunc()).
			WithURL(ws.URL + "/").WithBatchSize(10),
		ctxOK:  context.WithValue(context.Background(), "TK", "CALL OK"),
		ctx400: context.WithValue(context.Background(), "TK", "FAIL 400"),
		ctx401: context.WithValue(context.Background(), "TK", "FAIL 401"),
	}

	t.Run("call", ct.testService_Call)
	t.Run("objectlist", ct.testService_ObjectList)
	t.Run("describe", ct.testService_Describe)
	t.Run("getdeleted", ct.testService_GetDeleted)
	t.Run("getupdated", ct.testService_GetUpdated)
	t.Run("create", ct.testService_Create)
	t.Run("update", ct.testService_Update)
	t.Run("upsert", ct.testService_Upsert)
	t.Run("delete", ct.testService_Delete)
	t.Run("get", ct.testService_Get)
	t.Run("getByExternalID", ct.testService_GetByExternalID)
	t.Run("getattachment", ct.testService_GetAttachment)
	t.Run("createjob", ct.testService_CreateJob)
	t.Run("updatejobdata", ct.testService_UploadJobData)
	t.Run("closejob", ct.testService_CloseJob)
	t.Run("abortjob", ct.testService_AbortJob)
	t.Run("deletejob", ct.testService_DeleteJob)
	t.Run("getjob", ct.testService_GetJob)
	t.Run("getsuccessfuljobrecords", ct.testService_GetSuccessfulJobRecords)
	t.Run("getfailedjobrecords", ct.testService_GetFailedJobRecords)
	t.Run("getfailedjobrecords", ct.testService_GetUnprocessedJobRecords)
	t.Run("listjobs", ct.testService_ListJobs)
	t.Run("querycreatejob", ct.testService_QueryCreateJob)
	t.Run("retrieverecords", ct.testService_RetrieveRecords)
}

func getTokenClientFunc() ctxclient.Func {
	return func(ctx context.Context) (*http.Client, error) {
		tk, _ := ctx.Value("TK").(string)
		if tk == "" {
			return nil, errors.New("empty token")
		}
		return oauth2.Client(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tk}), nil), nil
	}
}

func testSrvGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var getAcct = Account{
		AccountName:  "Acct Name",
		AccountType:  "Business",
		RecordTypeID: "AAAAAAAAAAAAA",
		VendorID:     "VN12345",
	}
	switch r.URL.Path {

	case "/sobjects/":
		writeJSONFile(w, "testfiles/get/objects.json")
	case "/sobjects/Contact/describe":
		writeJSONFile(w, "testfiles/get/describe.json")
	case "/sobjects/Contact/deleted/":
		dateCheckResponse(w, "testfiles/get/getdeleted.json", r.URL.Query().Get("start"), r.URL.Query().Get("end"))
	case "/sobjects/Contact/updated/":
		dateCheckResponse(w, "testfiles/get/getupdated.json", r.URL.Query().Get("start"), r.URL.Query().Get("end"))
	case "/sobjects/Account/a1f4S000000cj9mQAA":
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(getAcct)
	case "/sobjects/Account/Vendor_ID__c/VN12345":
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(getAcct)
	case "/sobjects/Attachment/att4S000000cj9mQAA":
		writeFile(w, "testfiles/get/original.jpg", "image/jpeg")
	default:
		testSrvGetIngest(w, r.URL.Path)
	}

}

func testSrvGetIngest(w http.ResponseWriter, path string) {
	switch path {
	case "/jobs/ingest/JOB0000":
		encodeObject(w, salesforce.Job{
			ID:                  "JOB0000",
			APIVersion:          53,
			ColumnDelimiter:     "COMMA",
			ContentType:         "CSV",
			ExternalIDFieldName: "Vendor_ID__c",
			Object:              "Account",
			State:               "JobComplete",
		})
	case "/jobs/ingest/JOB0000/successfulResults/":
		writeFile(w, "testfiles/get/successrecords.csv", "test/csv")
	case "/jobs/ingest/JOB0000/failedResults/":
		writeFile(w, "testfiles/get/failedrecords.csv", "test/csv")
	case "/jobs/ingest/JOB0000/unprocessedrecords/":
		writeFile(w, "testfiles/get/unprocessedrecords.csv", "test/csv")
	case "/jobs/ingest/":
		encodeObject(w, salesforce.JobList{
			NextRecordsURL: path + "nextrecords",
			Records: []salesforce.Job{
				{
					ID:                  "JOB0000",
					APIVersion:          53,
					ColumnDelimiter:     "COMMA",
					ContentType:         "CSV",
					ExternalIDFieldName: "Vendor_ID__c",
					Object:              "Account",
					State:               "JobComplete",
				},
				{
					ID:                  "JOB0001",
					APIVersion:          53,
					ColumnDelimiter:     "COMMA",
					ContentType:         "CSV",
					ExternalIDFieldName: "Vendor_ID__c",
					Object:              "Account",
					State:               "Open",
				},
			},
		})
	case "/jobs/ingest/nextrecords":
		encodeObject(w, salesforce.JobList{
			Done: true,
			Records: []salesforce.Job{
				{
					ID:                  "JOB0000",
					APIVersion:          53,
					ColumnDelimiter:     "COMMA",
					ContentType:         "CSV",
					ExternalIDFieldName: "Vendor_ID__c",
					Object:              "Account",
					State:               "JobComplete",
				},
				{
					ID:                  "JOB0001",
					APIVersion:          53,
					ColumnDelimiter:     "COMMA",
					ContentType:         "CSV",
					ExternalIDFieldName: "Vendor_ID__c",
					Object:              "Account",
					State:               "Open",
				},
			},
		})
	default:
		http.Error(w, "not found", 404)
	}

}

func dateCheckResponse(w http.ResponseWriter, filename, sdt, edt string) {
	_, errs := time.Parse(time.RFC3339, sdt)
	_, erre := time.Parse(time.RFC3339, edt)
	if errs != nil || erre != nil {
		http.Error(w, "unable to parse date args", http.StatusBadRequest)
		return
	}
	writeJSONFile(w, filename)
}

func testSrvPatch(c context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sobjects/Account/a1f4S000000cj9mQAA":
		w.WriteHeader(204)
		return
	case "/sobjects/Account/Vendor_ID__c/VN12345":
		writeJSONFile(w, "testfiles/patch/updatedok.json")
		return
	case "/sobjects/Account/Vendor_ID__c/VN12346":
		w.WriteHeader(201)
		writeJSONFile(w, "testfiles/post/createdok.json")
		return
	case "/jobs/ingest/JOB0000":
		var args = make(map[string]string)
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
			return
		}
		encodeObject(w, salesforce.Job{
			ID:                  "JOB0000",
			APIVersion:          53,
			ColumnDelimiter:     "COMMA",
			ContentType:         "CSV",
			ExternalIDFieldName: "Vendor_ID__c",
			Object:              "Account",
			State:               args["state"],
		})
		return
	}
}

func testSrvDelete(c context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sobjects/Account/a1f4S000000cj9mQAA":
		w.WriteHeader(204)
		return
	case "/jobs/ingest/JOB0000":
		w.WriteHeader(204)
		return
	}
}

func testSrvPut(c context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/jobs/ingest/JOB0000/batches":
		defer r.Body.Close()
		cr := csv.NewReader(r.Body)
		rows, err := cr.ReadAll()
		if err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}
		if len(rows) != 5 {
			http.Error(w, fmt.Sprintf("expected 4 rows + header; got %v", err), http.StatusInternalServerError)
		}
		return
	}
}

func testSrvPost(c context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sobjects/Account":
		w.WriteHeader(201)
		writeJSONFile(w, "testfiles/post/createdok.json")
		return
	case "/jobs/ingest/":
		encodeObject(w, salesforce.Job{
			ID:                  "JOB0000",
			APIVersion:          53,
			ColumnDelimiter:     "COMMA",
			ContentType:         "CSV",
			ExternalIDFieldName: "Vendor_ID__c",
			Object:              "Account",
			State:               "Open",
		})
		return
	case "/jobs/query":
		encodeObject(w, salesforce.Job{
			ID:              "JOB0006",
			Operation:       "queryAll",
			APIVersion:      53,
			ColumnDelimiter: "COMMA",
			ContentType:     "CSV",
			Object:          "Account",
			State:           "UploadComplete",
		})
		return
	case "/composite/sobjects/Contact":
		encodeObject(w, []Contact{
			{
				ContactID:   "0033000002239QCA",
				AccountID:   "0013000008020XAB",
				FirstName:   "FirstForename",
				ExternalPID: "P000297",
			},
			{
				ContactID:   "003300000223aQCA",
				AccountID:   "0013000008021XAB",
				FirstName:   "David",
				ExternalPID: "P000298",
			},
			{
				ContactID:   "003300000223bQCA",
				AccountID:   "0013000008022XAB",
				FirstName:   "John",
				ExternalPID: "P000299",
			},
			{
				ContactID:   "003300000223cQCA",
				AccountID:   "0013000008023XAB",
				FirstName:   "Paul",
				ExternalPID: "P00029a",
			},
		})
	}

}

func serviceHandlerFunc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()
	if !checkAuth(w, strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)) {
		return
	}
	if strings.HasSuffix(r.URL.Path, "/withtoken") {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer WITHTOKEN" {
			http.Error(w, "expectd WITHTOKEN; got "+auth, http.StatusUnauthorized)
		}
		return
	}

	switch r.Method {
	case "GET":
		testSrvGet(ctx, w, r)
		return
	case "PATCH":
		testSrvPatch(ctx, w, r)
		return
	case "DELETE":
		testSrvDelete(ctx, w, r)
		return
	case "PUT":
		testSrvPut(ctx, w, r)
		return
	case "POST":
		testSrvPost(ctx, w, r)
		return
	}
	http.Error(w, fmt.Sprintf("invalid path %s", r.URL.Path), http.StatusNotFound)
}

var authErr = errors.New("auth error")

func checkAuth(w http.ResponseWriter, tk string) bool {
	if tk == "CALL OK" || tk == "WITHTOKEN" {
		return true
	}
	status := http.StatusInternalServerError
	params := strings.Split(tk, " ")
	if len(params) > 1 {
		if status, _ = strconv.Atoi(params[len(params)-1]); status == 0 {
			status = http.StatusInternalServerError
		}
	}
	http.Error(w, tk, status)
	return false
}

func writeJSONFile(w http.ResponseWriter, fn string) bool {
	return writeFile(w, fn, "application/json")
}

func writeFile(w http.ResponseWriter, fn, mimeType string) bool {
	wr := w
	f, err := os.Open(fn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	defer f.Close()
	w.Header().Set("Content-type", mimeType)
	if _, err = io.Copy(wr, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func encodeObject(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		http.Error(w, fmt.Sprintf("json %v", err), http.StatusInternalServerError)
	}
	return
}

func TestDeleteID(t *testing.T) {
	var val salesforce.DeleteID = "DELID"

	if val.SObjectName() != "DeleteID" {
		t.Errorf("expecte SObjectName of DeleteID; got %s", val.SObjectName())
		return
	}

	val2 := val.WithAttr("")
	if val2 != val {
		t.Errorf("WithAttribute should not change value")
	}
}

func TestOpResponse_SObjectValue(t *testing.T) {

	var contactPtr = &Contact{}

	tests := []struct {
		name    string
		ix      interface{}
		or      salesforce.OpResponse
		wantErr bool
	}{
		{name: "t00", wantErr: true},
		{name: "t01", ix: 5, wantErr: true},
		{name: "t02", or: salesforce.OpResponse{SObject: CustomTable{}}, ix: contactPtr, wantErr: true},
		{name: "t03", or: salesforce.OpResponse{SObject: Contact{}}, ix: contactPtr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := tt.or.SObjectValue(tt.ix); (err != nil) != tt.wantErr {
				t.Errorf("OpResponse.SObjectValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
