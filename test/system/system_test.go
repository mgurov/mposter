package system_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/mgurov/mposter/internal/assertions"
	"github.com/mgurov/mposter/internal/testserver"
)

func TestSimpleCalling(t *testing.T) {

	server := testserver.StartNewTestServer()
	defer server.Shutdown()

	input := `A
B
C
`
	command := "mposter --tick=2 " + server.Addr() + "/path/"

	runResult := runWithErr(command, input, t)

	assertions.OnlyLinesContaining(t, "errstr", []string{
		"2 ERR: 0",
		"Done 3 OK: 3 ERR: 0",
	}, runResult.stdErr.String())

	if runResult.exitCode != nil {
		t.Error("Unexpected error code", runResult.exitCode.ExitCode())
	}

	assertions.StringEqual(t, "stdout", "A OK\nB OK\nC OK\n", runResult.stdOut.String())

	expectedLog := `POST /path/A
POST /path/B
POST /path/C
`

	assertions.StringEqual(t, "http access log", expectedLog, server.AccessLog())
}

func TestUnknownFlag(t *testing.T) {

	runResult := runWithErr("mposter --unknown-flag", "", t)

	if runResult.exitCode == nil {
		t.Error("expected non-zero exit code but got none")
	}

	expectedStdErr := "flag provided but not defined: -unknown-flag"
	if !strings.Contains(runResult.stdErr.String(), expectedStdErr) {
		t.Errorf("expected error message to contain \"%s\" got: %s", expectedStdErr, runResult.stdErr.String())
	}
}

func run(command, input string, t *testing.T) string {
	runResult := runWithErr(command, input, t)

	if runResult.stdErr.String() != "" {
		t.Error("Unexpected error output:", runResult.stdErr.String())
	}
	if runResult.exitCode != nil {
		t.Error("Unexpected error code", runResult.exitCode.ExitCode())
	}

	return runResult.stdOut.String()
}

func runWithErr(command, input string, t *testing.T) runResultType {

	var result runResultType

	cmd := exec.Command("sh", "-c", "../../build/out/"+command)
	cmd.Stdout = &result.stdOut
	cmd.Stderr = &result.stdErr
	cmd.Stdin = bytes.NewBufferString(input)

	err := cmd.Run()

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			result.exitCode = ee
		} else {
			t.Error("Unexpected error: ", err)
		}
	}

	return result
}

type runResultType struct {
	stdErr   bytes.Buffer
	stdOut   bytes.Buffer
	exitCode *exec.ExitError
}
