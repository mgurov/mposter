package system_test

import (
	"bytes"
	"fmt"
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
	output := run("mposter --url "+server.Addr()+"/path/", input)

	assertions.StringEqual(t, "stdout", "A OK\nB OK\nC OK\n", output)

	expectedLog := `POST /path/A
POST /path/B
POST /path/C
`

	assertions.StringEqual(t, "http access log", expectedLog, server.AccessLog())
}

func TestUnknownFlag(t *testing.T) {

	runResult := runWithErr("mposter --unknown-flag", "")

	if runResult.exitCode == nil {
		t.Error("expected non-zero exit code but got none")
	}

	expectedStdErr := "flag provided but not defined: -unknown-flag"
	if !strings.Contains(runResult.stdErr.String(), expectedStdErr) {
		t.Errorf("expected error message to contain \"%s\" got: %s", expectedStdErr, runResult.stdErr.String())
	}
}

func run(command, input string) string {
	runResult := runWithErr(command, input)

	if runResult.stdErr.String() != "" {
		fmt.Println("Unexpected error output:", runResult.stdErr.String())
	}
	if runResult.exitCode != nil {
		fmt.Println("Unexpected error code", runResult.exitCode.ExitCode())
	}

	return runResult.stdOut.String()
}

func runWithErr(command, input string) runResultType {

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
			fmt.Println("Unexpected error: ", err)
		}
	}

	return result
}

type runResultType struct {
	stdErr   bytes.Buffer
	stdOut   bytes.Buffer
	exitCode *exec.ExitError
}
