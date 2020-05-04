package assertions

import (
	"strings"
	"testing"
)

func ErrorContains(t *testing.T, expectedSub string, actual error) {
	if nil == actual || !strings.Contains(actual.Error(), expectedSub) {
		t.Errorf("expected err containing: '%s' got: '%q' ", expectedSub, actual)
	}
}

func NoError(t *testing.T, actual error) {
	if nil != actual {
		t.Errorf("expected no err got: '%q' ", actual)
	}
}
