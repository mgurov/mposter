package urltemplate

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {

	type parsedTest struct {
		name              string
		rowInput          []string
		want              string
		wantErrContaining string
	}

	tests := []struct {
		input       string
		parsedTests []parsedTest
	}{
		{
			input: "blah{{0}}fooe{{1}}zooe",
			parsedTests: []parsedTest{
				{
					name:     "plain",
					rowInput: []string{"0", "1"},
					want:     "blah0fooe1zooe",
				},
				{
					name:     "spaced",
					rowInput: []string{"0 1", "1 2"},
					want:     "blah0 1fooe1 2zooe",
				},
				{
					name:              "err on missing param",
					rowInput:          []string{"0"},
					wantErrContaining: "data missing for placeholder {{1}}",
				},
				{
					name:     "no problem on extra param",
					rowInput: []string{"0", "1", "2"},
					want:     "blah0fooe1zooe",
				},
			},
		},
		{
			input: "blah{0}",
			parsedTests: []parsedTest{
				{
					name:     "params ignored since not parsed",
					rowInput: []string{"a", "b"},
					want:     "blah{0}",
				},
				{
					name:     "empty params are also ok even won't happen",
					rowInput: []string{},
					want:     "blah{0}",
				},
			},
		},
		{
			input: "blah { {0}}",
			parsedTests: []parsedTest{
				{
					name:     "params ignored since not parsed",
					rowInput: []string{"a", "b"},
					want:     "blah { {0}}",
				},
			},
		},
		{
			input: "blah{{ 0 }}fooe",
			parsedTests: []parsedTest{
				{
					name:     "spaces removed",
					rowInput: []string{"a", "b"},
					want:     "blahafooe",
				},
			},
		},
	}

	for _, t1 := range tests {
		t.Run(t1.input, func(t *testing.T) {
			gotParsed, err := Parse(t1.input)
			if err != nil {
				t.Errorf("Parse() error = %v, wantErr no err", err)
				return
			}

			for _, tt := range t1.parsedTests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := gotParsed(tt.rowInput)
					if err != nil {
						if "" == tt.wantErrContaining {
							t.Errorf("apply parsed error = %v, want no Err", err)
						} else if !strings.Contains(err.Error(), tt.wantErrContaining) {
							t.Errorf("apply parsed error = %v, wantErr containing '%s'", err, tt.wantErrContaining)
						}
						return
					}
					if got != tt.want {
						t.Errorf("apply parsed = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}
func TestParseErr(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		wantErrContaining string
	}{
		{
			name:              "unclosed",
			input:             "blah{{0",
			wantErrContaining: "placeholder '{{0' isn't terminated",
		},
		{
			name:              "partially unclosed",
			input:             "blah{{0}",
			wantErrContaining: "placeholder '{{0}' isn't terminated",
		},
		{
			name:              "unrecognized content",
			input:             "blah{{fooe}}",
			wantErrContaining: "placeholder '{{fooe}}' isn't recognized",
		},
		{
			name:              "empty placeholder unrecognized",
			input:             "{{}}",
			wantErrContaining: "placeholder '{{}}' isn't recognized",
		},
		{
			name:              "placeholder start within placeholder unrecognized",
			input:             "{{a {{ }}",
			wantErrContaining: "placeholder '{{a {{ }}' isn't recognized",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)

			if err == nil {
				t.Errorf("Parse() want error")
			} else if !strings.Contains(err.Error(), tt.wantErrContaining) {
				t.Errorf("Parse() = err %v but want err containing %s", err, tt.wantErrContaining)
			}
		})
	}
}
