package runparams

import (
	"testing"

	"github.com/mgurov/mposter/internal/assertions"
)

func TestParseFieldSeparator(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "default",
			args: []string{"url"},
			want: "",
		},
		{
			name: "overwrite",
			args: []string{"-separator", ",", "url"},
			want: ",",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse("", tt.args)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			if got.FieldSeparator != tt.want {
				t.Errorf("Parse() = %s, want %v", got.FieldSeparator, tt.want)
			}
		})
	}

}

func TestParseUnknownError(t *testing.T) {
	_, err := Parse("", []string{"--unknown-flag"})

	assertions.ErrorContains(t, "flag provided but not defined: -unknown-flag", err)

}
func TestParseUrl(t *testing.T) {
	parsed, err := Parse("", []string{"--dry-run", "https://host:port/path/"})

	assertions.NoError(t, err)

	assertions.StringEqual(t, "Url", "https://host:port/path/", parsed.Url)
}

func TestParseUrl_ShouldFailOnMultiple(t *testing.T) {
	_, err := Parse("", []string{"--dry-run", "https://host:port/path/", "something else"})

	assertions.ErrorContains(t, "multiple urls provided", err)
}

func TestParseUrl_ShouldFailOnNotProvided(t *testing.T) {
	_, err := Parse("", []string{"--dry-run"})

	assertions.ErrorContains(t, "url not provided", err)
}

func TestParseUrl_ShouldSupportFurtherParsing(t *testing.T) {
	parsed, err := Parse("", []string{"--http-method", "before url", "url", "--separator", "after url"})

	assertions.NoError(t, err)
	assertions.StringEqual(t, "Url", "url", parsed.Url)
	assertions.StringEqual(t, "HttpMethod", "before url", parsed.HttpMethod)
	assertions.StringEqual(t, "Separator", "after url", parsed.FieldSeparator)
}
