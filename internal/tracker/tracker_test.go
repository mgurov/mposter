package tracker

import (
	"testing"

	"github.com/mgurov/mposter/internal/assertions"
)

func Test_StopExecutionOnFirstError(t *testing.T) {

	tests := []struct {
		name     string
		template Tracker
	}{
		{
			name:     "default no stop consecutive",
			template: Tracker{StopOnConsecutiveErrCount: 0},
		},
		{
			name:     "stop on 2 consecutive", //one consecutive would stop this or another way
			template: Tracker{StopOnConsecutiveErrCount: 2},
		},
		{
			name:     "stop on gazillion consecutive",
			template: Tracker{StopOnConsecutiveErrCount: 198765},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/on-first-ok", func(t *testing.T) {
			firstOkCopy := tt.template
			firstOkCopy.StopOnFirstErr = true
			firstOkCopy.Ok()
			assertions.NoError(t, firstOkCopy.Err())
		})
		t.Run(tt.name+"/on-first-err", func(t *testing.T) {
			firstErrCopy := tt.template
			firstErrCopy.StopOnFirstErr = true
			assertions.ErrorContains(t, "error on first call", firstErrCopy.Err())
		})
		t.Run(tt.name+"/off-first-ok", func(t *testing.T) {
			firstOkCopy := tt.template
			firstOkCopy.StopOnFirstErr = false
			firstOkCopy.Ok()
			assertions.NoError(t, firstOkCopy.Err())
		})
		t.Run(tt.name+"/off-first-err", func(t *testing.T) {
			firstErrCopy := tt.template
			firstErrCopy.StopOnFirstErr = false
			assertions.NoError(t, firstErrCopy.Err())
		})
	}
}

func Test_StopExecutionOnConsecutiveErrorNumber(t *testing.T) {
	testee := Tracker{StopOnConsecutiveErrCount: 2}

	assertions.NoError(t, testee.Err())
	assertions.ErrorContains(t, "2 consecutive errors", testee.Err())
}

func Test_StopExecutionOnConsecutiveErrorNumber_reset(t *testing.T) {
	testee := Tracker{StopOnConsecutiveErrCount: 2}

	assertions.NoError(t, testee.Err())
	testee.Ok()
	assertions.NoError(t, testee.Err())
	assertions.ErrorContains(t, "2 consecutive errors", testee.Err())
}

func Test_StopExecutionOnConsecutiveErrorNumber_disabled(t *testing.T) {
	testee := Tracker{StopOnConsecutiveErrCount: 0}

	assertions.NoError(t, testee.Err())
	assertions.NoError(t, testee.Err())
	assertions.NoError(t, testee.Err())
}

func Test_StopExecutionOnConsecutiveErrorNumber_one(t *testing.T) {
	testee := Tracker{StopOnConsecutiveErrCount: 1}

	testee.Ok()
	assertions.ErrorContains(t, "1 consecutive errors", testee.Err())
}
