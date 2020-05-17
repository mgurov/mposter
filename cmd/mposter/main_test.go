package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mgurov/mposter/cmd/mposter/runparams"
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
		run.runParams.Url = "http://localhost/"
		run.runParams.DryRun = true
	})
	result.AssertHttpAccessLog("")
	result.AssertOutput("A POST http://localhost/A\nB POST http://localhost/B\n")
}

func TestDryRunSpaces(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = " A \nB"
		run.runParams.Url = "http://localhost/"
		run.runParams.DryRun = true
	})
	result.AssertHttpAccessLog("")
	result.AssertOutput("A POST http://localhost/A\nB POST http://localhost/B\n")
}

func TestDryRunSpacesAroundCommas(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = " A , B \n C , D "
		run.runParams.Url = "http://localhost/{{0}}/sub/{{1}}"
		run.runParams.DryRun = true
		run.runParams.FieldSeparator = ","
	})
	result.AssertHttpAccessLog("")
	result.AssertOutput("A , B POST http://localhost/A/sub/B\nC , D POST http://localhost/C/sub/D\n")
}
func TestDryRunOtherVerb(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB"
		run.runParams.Url = "http://localhost/"
		run.runParams.DryRun = true
		run.runParams.HttpMethod = "DELETE"
	})
	result.AssertHttpAccessLog("")
	result.AssertOutput("A DELETE http://localhost/A\nB DELETE http://localhost/B\n")
}

func TestSkipFirstLines(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "HEADER\nB\nC"
		run.runParams.Skip = 1

	})
	result.AssertHttpAccessLog("POST /B\n" +
		"POST /C\n")
	result.AssertOutput("B OK\nC OK\n")
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
		run.path = "/path/{{0}}/sub/{{1}}"
	})

	expectedLog := "POST /path/A/sub/1\n" +
		"POST /path/B/sub/2\n" +
		"POST /path/C/sub/3\n"

	assertions.StringEqual(t, "http log", expectedLog, result.ActualServerAccess())
}
func TestMultipleParametersSupportCommaSeparated(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A,1\nB,2\nC,3"
		run.path = "/path/{{0}}/sub/{{1}}"
		run.runParams.FieldSeparator = ","
	})

	result.AssertHttpAccessLog("POST /path/A/sub/1\n" +
		"POST /path/B/sub/2\n" +
		"POST /path/C/sub/3\n")
}

func TestShouldTrimSpacesWhenParameterized(t *testing.T) {
	result := execute(t, func(run *TestRun) {
		run.input = "A \n B "
		run.path = "/path/{{0}}/sub/"
	})

	expectedLog := "POST /path/A/sub/\n" +
		"POST /path/B/sub/\n"

	assertions.StringEqual(t, "http log", expectedLog, result.ActualServerAccess())
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
		run.runParams.StopOnErrorCount = 2
		run.errCheck = ExpectErrContaining("2 consecutive errors")
	})

	result.AssertOutput("A OK\nfail ERR HTTP 500\nfail ERR HTTP 500\n")
}

func TestShouldNotStopOnNonConsecutiveErrors(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nfail\nB\nfail\nC"
		run.server.ReturnEmptyResponseWithHttpStatus("/fail", 500)
		run.runParams.StopOnErrorCount = 2
	})

	result.AssertOutput("A OK\nfail ERR HTTP 500\nB OK\nfail ERR HTTP 500\nC OK\n")
}

func TestShouldStopAtOnceOnFirstError(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "fail\nA"
		run.server.ReturnEmptyResponseWithHttpStatus("/fail", 500)
		run.runParams.StopOnErrorCount = 2
		run.runParams.StopOnFirstError = true
		run.errCheck = ExpectErrContaining("error on first call")
	})

	result.AssertOutput("fail ERR HTTP 500\n")
}

func TestShouldStopAtOnceOnFirstTimeout(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "delay\nA"
		run.runParams.Timeout = 10 * time.Millisecond
		run.server.RegisterHandler("/delay", DelayResponseHandler(20*time.Millisecond))
		run.runParams.StopOnErrorCount = 2
		run.runParams.StopOnFirstError = true
		run.errCheck = ExpectErrContaining("error on first call")
	})

	result.AssertOutput("delay ERR Timeout\n")
}

func TestShouldTimeoutOnTimeout(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nlongB\nC"
		run.runParams.Timeout = 10 * time.Millisecond
		run.server.RegisterHandler("/longB", DelayResponseHandler(20*time.Millisecond))
		run.runParams.StopOnErrorCount = 2
	})

	result.AssertOutput("A OK\nlongB ERR Timeout\nC OK\n")
}

func TestAlternativeHttpVerb(t *testing.T) {

	result := execute(t, func(run *TestRun) {
		run.input = "A\nB"
		run.runParams.HttpMethod = "DELETE"
	})
	result.AssertHttpAccessLog("DELETE /A\n" +
		"DELETE /B\n")
	result.AssertOutput("A OK\nB OK\n")

}

func whenRan(t *testing.T, input, path string) string {
	return whenRanWithParams(t, input, path, func(it runparams.RunParams) runparams.RunParams { return it })
}

func whenRanWithParams(t *testing.T, input, path string, paramsFun func(runparams.RunParams) runparams.RunParams) string {
	server := testserver.StartNewTestServer()
	defer server.Shutdown()

	runParams := runparams.NewRunParams()
	runParams.Url = server.Addr() + path
	runParams.Input = strings.NewReader(input)
	runParams.Output = ioutil.Discard

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
	runParams    runparams.RunParams
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
	tr.runParams.Output = &tr.actualOutput
	tr.runParams.HttpMethod = "POST"

	adjuster(&tr)

	tr.server.Start()
	defer tr.server.Shutdown()

	if tr.runParams.Url == "" {
		tr.runParams.Url = tr.server.Addr() + tr.path
	}

	if tr.runParams.Input == nil {
		tr.runParams.Input = strings.NewReader(tr.input)
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
