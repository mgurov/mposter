package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/mgurov/mposter/internal/assertions"
	"github.com/mgurov/mposter/internal/testserver"
)

func TestSimpleRun(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB\nC"
		run.path = "/path/"
	})
	result.AssertHttpAccessLog("POST /path/A\n" +
		"POST /path/B\n" +
		"POST /path/C\n")
	result.AssertOutput("A OK\nB OK\nC OK\n")

}

func TestShouldSkipEmptyLines(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "\nA\n\nB\nC\n"
	})
	result.AssertHttpAccessLog("POST /A\nPOST /B\nPOST /C\n")
	result.AssertOutput("A OK\nB OK\nC OK\n")
}

func TestShouldPathEncode(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "a#b+c"
		run.path = "/path/"
	})

	result.AssertHttpAccessLog("POST /path/a%23b+c\n")

	result.AssertOutput("a#b+c OK\n")
}

func TestShouldQueryEncode(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "a#b+c"
		run.path = "/path?query="
	})

	result.AssertHttpAccessLog("POST /path?query=a%23b%2Bc\n")

	result.AssertOutput("a#b+c OK\n")
}

func TestMultipleParametersSupport(t *testing.T) {
	result := execute(t, func(run *TestRun) {
		run.input = "A 1\nB 2\nC 3"
		run.path = "/path/{{index . 0}}/sub/{{index . 1}}"
	})

	expectedLog := "POST /path/A/sub/1\n" +
		"POST /path/B/sub/2\n" +
		"POST /path/C/sub/3\n"

	assertions.StringEqual(t, "http log", expectedLog, result.ActualServerAccess())
}
func TestMultipleParametersSupportCommaSeparated(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A,1\nB,2\nC,3"
		run.path = "/path/{{index . 0}}/sub/{{index . 1}}"
		run.runParams.fieldSeparator = ","
	})

	result.AssertHttpAccessLog("POST /path/A/sub/1\n" +
		"POST /path/B/sub/2\n" +
		"POST /path/C/sub/3\n")
}

func TestShouldReportNon200Statuses(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB\nC"
		run.server.ReturnEmptyResponseWithHttpStatus("/B", 500)
		run.server.ReturnEmptyResponseWithHttpStatus("/C", 404)
	})

	result.AssertHttpAccessLog("POST /A\nPOST /B\nPOST /C\n")
	result.AssertOutput("A OK\nB ERR HTTP 500\nC ERR HTTP 404\n")
}

func TestShouldContinueOnError(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB\nC"
		run.server.ReturnEmptyResponseWithHttpStatus("/B", 500)
	})

	result.AssertHttpAccessLog("POST /A\nPOST /B\nPOST /C\n")
	result.AssertOutput("A OK\nB ERR HTTP 500\nC OK\n")
}

func TestShouldStopOnConsecutiveErrors(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nfail\nfail\nD"
		run.server.ReturnEmptyResponseWithHttpStatus("/fail", 500)
		run.runParams.stopOnErrorCount = 2
		run.suppressNoErrCheck = true
	})

	result.AssertOutput("A OK\nfail ERR HTTP 500\nfail ERR HTTP 500\n")
}

func TestShouldNotStopOnNonConsecutiveErrors(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nfail\nB\nfail\nC"
		run.server.ReturnEmptyResponseWithHttpStatus("/fail", 500)
		run.runParams.stopOnErrorCount = 2
	})

	result.AssertOutput("A OK\nfail ERR HTTP 500\nB OK\nfail ERR HTTP 500\nC OK\n")
}

func whenRan(t *testing.T, input, path string) string {
	return whenRanWithParams(t, input, path, func(it runParams) runParams { return it })
}

func whenRanWithParams(t *testing.T, input, path string, paramsFun func(runParams) runParams) string {
	server := testserver.StartNewTestServer()
	defer server.Close()

	runParams := paramsFun(runParams{
		url:    server.Addr() + path,
		input:  strings.NewReader(input),
		output: ioutil.Discard,
	})

	err := run(runParams)
	if err != nil {
		t.Error("Run failed", err)
	}

	return server.AccessLog()
}

type TestRun struct {
	t                  *testing.T
	input              string
	path               string
	suppressNoErrCheck bool
	runParams          runParams
	server             *testserver.TestServer
	actualOutput       bytes.Buffer
	actualErr          error
}

func (tr TestRun) ActualServerAccess() string {
	return tr.server.AccessLog()
}

func (tr TestRun) ActualOutput() string {
	return tr.actualOutput.String()
}

func (tr TestRun) AssertHttpAccessLog(expectedLog string) {
	assertions.StringEqual(tr.t, "http access log", expectedLog, tr.ActualServerAccess())
}

func (tr TestRun) AssertOutput(expectedOutput string) {
	assertions.StringEqual(tr.t, "output", expectedOutput, tr.ActualOutput())
}

func execute(t *testing.T, adjuster func(*TestRun)) *TestRun {

	s := testserver.NewTestServer()

	tr := TestRun{
		server: &s,
		path:   "/",
		t:      t,
	}

	tr.runParams.output = &tr.actualOutput

	adjuster(&tr)

	tr.server.Start()
	defer tr.server.Close()

	if tr.runParams.url == "" {
		tr.runParams.url = tr.server.Addr() + tr.path
	}

	if tr.runParams.input == nil {
		tr.runParams.input = strings.NewReader(tr.input)
	}

	tr.actualErr = run(tr.runParams)
	if tr.actualErr != nil && !tr.suppressNoErrCheck {
		t.Error("Run failed with err:", tr.actualErr)
	}

	return &tr
}
