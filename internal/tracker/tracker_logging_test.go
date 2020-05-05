package tracker

import (
	"bytes"
	"log"
	"testing"

	"github.com/mgurov/mposter/internal/assertions"
)

func Test_Log(t *testing.T) {

	capturedOutput := bytes.Buffer{}

	testee := Tracker{
		Logger:  log.New(&capturedOutput, "", 0),
		TickLog: 1,
	}

	//when
	testee.Ok()
	testee.Ok()
	testee.Err()
	testee.LogDone()

	expectedOutput := `1 ERR: 0
2 ERR: 0
3 ERR: 1
Done 3 OK: 2 ERR: 1
`

	assertions.StringEqual(t, "", expectedOutput, capturedOutput.String())
}

func Test_LogFirstErr(t *testing.T) {

	capturedOutput := bytes.Buffer{}

	testee := Tracker{
		Logger:      log.New(&capturedOutput, "", 0),
		LogFirstErr: true,
		TickLog:     100,
	}

	//when
	testee.Ok()
	testee.Err()
	testee.Err()
	testee.LogDone()

	expectedOutput := `2 ERR: 1
Done 3 OK: 1 ERR: 2
`
	assertions.StringEqual(t, "", expectedOutput, capturedOutput.String())
}

func Test_Log_Not_When_Tick_0(t *testing.T) {

	capturedOutput := bytes.Buffer{}

	testee := Tracker{
		Logger:  log.New(&capturedOutput, "", 0),
		TickLog: 0,
	}

	//when
	testee.Ok()
	testee.Ok()
	testee.Err()
	testee.LogDone()

	assertions.StringEqual(t, "", "Done 3 OK: 2 ERR: 1\n", capturedOutput.String())
}

func Test_ShouldNotLogWhenLoggerIsNull(t *testing.T) {
	testee := Tracker{Logger: nil}

	testee.Ok()
	testee.Err()
}
