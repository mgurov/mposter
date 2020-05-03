package assertions

import "testing"

func StringEqual(t *testing.T, title, expected, actual string) {
	if expected != actual {
		t.Errorf("%s expected: '%s' got: '%s' ", title, expected, actual)
	}

}
