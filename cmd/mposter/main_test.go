package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

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
func TestDryRun(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB"
		run.runParams.url = "http://localhost/"
		run.runParams.dryRun = true
	})
	result.AssertHttpAccessLog("")
	result.AssertOutput("A POST http://localhost/A\nB POST http://localhost/B\n")
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
		run.errCheck = ExpectErrContaining("2 consecutive errors")
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

func TestShouldStopAtOnceOnFirstError(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "fail\nA"
		run.server.ReturnEmptyResponseWithHttpStatus("/fail", 500)
		run.runParams.stopOnErrorCount = 2
		run.runParams.stopOnFirstError = true
		run.errCheck = ExpectErrContaining("error on first call")
	})

	result.AssertOutput("fail ERR HTTP 500\n")
}

func TestShouldStopAtOnceOnFirstTimeout(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "delay\nA"
		run.runParams.timeout = 10 * time.Millisecond
		run.server.RegisterHandler("/delay", DelayResponseHandler(20*time.Millisecond))
		run.runParams.stopOnErrorCount = 2
		run.runParams.stopOnFirstError = true
		run.errCheck = ExpectErrContaining("error on first call")
	})

	result.AssertOutput("delay ERR Timeout\n")
}

func TestShouldTimeoutOnTimeout(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nlongB\nC"
		run.runParams.timeout = 10 * time.Millisecond
		run.server.RegisterHandler("/longB", DelayResponseHandler(20*time.Millisecond))
		run.runParams.stopOnErrorCount = 2
	})

	result.AssertOutput("A OK\nlongB ERR Timeout\nC OK\n")
}

func TestAlternativeHttpVerb(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB"
		run.runParams.httpMethod = "DELETE"
	})
	result.AssertHttpAccessLog("DELETE /A\n" +
		"DELETE /B\n")
	result.AssertOutput("A OK\nB OK\n")

}

func whenRan(t *testing.T, input, path string) string {
	return whenRanWithParams(t, input, path, func(it runParams) runParams { return it })
}

func whenRanWithParams(t *testing.T, input, path string, paramsFun func(runParams) runParams) string {
	server := testserver.StartNewTestServer()
	defer server.Shutdown()

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
	t            *testing.T
	input        string
	path         string
	errCheck     func(error, *testing.T)
	runParams    runParams
	server       *testserver.TestServer
	actualOutput bytes.Buffer
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
		errCheck: func(err error, tt *testing.T) {
			if nil != err {
				tt.Error("Unexpected err from run:", err)
			}
		},
	}

	//TODO: use default run params as a starting point
	tr.runParams.output = &tr.actualOutput
	tr.runParams.httpMethod = "POST"

	adjuster(&tr)

	tr.server.Start()
	defer tr.server.Shutdown()

	if tr.runParams.url == "" {
		tr.runParams.url = tr.server.Addr() + tr.path
	}

	if tr.runParams.input == nil {
		tr.runParams.input = strings.NewReader(tr.input)
	}

	actualErr := run(tr.runParams)

	tr.errCheck(actualErr, t)

	return &tr
}

func DelayResponseHandler(delay time.Duration) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(204)
	}
}

func ExpectErrContaining(sub string) func(error, *testing.T) {
	return func(err error, t *testing.T) {
		if err == nil || !strings.Contains(err.Error(), sub) {
			t.Errorf("Expected error containing %s but got %q", sub, err)
		}
	}
}
