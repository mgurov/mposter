package tracker

import (
	"fmt"
	"log"
)

//Tracker is *not* thread-safe.
type Tracker struct {
	rowNo                     int
	errCount                  int
	okCount                   int
	consecutiveErrCount       int
	StopOnFirstErr            bool
	StopOnConsecutiveErrCount int
	Logger                    *log.Logger
	TickLog                   int //number of messages to log the current status at
}

func (t *Tracker) Ok() {
	t.rowNo++
	t.okCount++
	t.consecutiveErrCount = 0
	t.maybeLogStatus()
}

// Err returns reason to bail out if such
func (t *Tracker) Err() error {
	t.rowNo++
	t.errCount++
	t.consecutiveErrCount++
	t.maybeLogStatus()

	if t.StopOnFirstErr && t.rowNo == 1 {
		return fmt.Errorf("error on first call")
	}
	if t.StopOnConsecutiveErrCount > 0 && t.consecutiveErrCount >= t.StopOnConsecutiveErrCount {
		return fmt.Errorf("%d consecutive errors", t.consecutiveErrCount)
	}
	return nil
}

func (t Tracker) maybeLogStatus() {
	if t.TickLog > 0 && t.rowNo%t.TickLog == 0 {
		t.LogStatus()
	}
}

func (t Tracker) LogStatus() {
	if nil != t.Logger {
		t.Logger.Printf("%d ERR: %d", t.rowNo, t.errCount)
	}
}

func (t Tracker) LogDone() {
	if nil != t.Logger {
		t.Logger.Printf("Done %d OK: %d ERR: %d", t.rowNo, t.okCount, t.errCount)
	}
}
