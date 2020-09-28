package assertions

import (
	"strings"
	"testing"
)

func StringEqual(t *testing.T, title, expected, actual string) {
	if expected != actual {
		t.Errorf("%s expected: '%s' got: '%s' ", title, expected, actual)
	}

}

func OnlyLinesContaining(t *testing.T, title string, expected []string, actual string) {

	expectedCounts := map[string]int{}
	for _, expectedLine := range expected {
		expectedCounts[expectedLine]++
	}

	t.Log("start expected lines", expectedCounts)

	unexpectedLines := []string{}

	actualLines := strings.Split(actual, "\n")
outside:
	for _, actualLine := range actualLines {
		if strings.TrimSpace(actualLine) == "" {
			continue
		}
		for expectedLine, expectedLineCount := range expectedCounts {
			if strings.Contains(actualLine, expectedLine) {
				expectedCounts[expectedLine] = expectedLineCount - 1
				continue outside
			}
		}
		unexpectedLines = append(unexpectedLines, actualLine)
	}

	t.Log("end expected lines", expectedCounts)

	for expectedLine, expectedLineCount := range expectedCounts {
		if expectedLineCount == 0 {
			continue
		}
		if expectedLineCount > 0 {
			t.Errorf("Unmet %d expectations for line containing %s (%s)", expectedLineCount, expectedLine, title)
		} else {
			t.Errorf("Overachieved %d expectations for line containing %s (%s)", -expectedLineCount, expectedLine, title)
		}
	}

	for _, unexpectedLine := range unexpectedLines {
		t.Errorf("Unexpected line %s (%s)", unexpectedLine, title)
	}

}
