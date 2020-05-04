package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
)

func main() {
	params, err := parseParams(os.Args)
	if nil != err {
		log.Fatal(err)
	}

	err = run(params)
	if nil != err {
		log.Fatal(err)
	}
}

func parseParams(params []string) (runParams, error) {

	//TODO: support reading to and writing from a file
	result := newRunParams()

	commandLine := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	commandLine.StringVar(&result.url, "url", "http://localhost:8080/example/", "url to post to") //TODO: no default and fail if not specified.
	commandLine.StringVar(&result.fieldSeparator, "separator", "", "row field separator. White space if not specified.")
	commandLine.IntVar(&result.stopOnErrorCount, "stop-on-err-count", result.stopOnErrorCount, "Stop on consequent error results")
	commandLine.BoolVar(&result.stopOnFirstError, "stop-on-first-err", true, "stop on very first error at once, disregarding the stop-on-err-count setting")
	commandLine.DurationVar(&result.timeout, "timeout", 0, "http timeout, 0 (default) meaning no timeout")

	return result, commandLine.Parse(os.Args[1:])
}

func run(params runParams) error {

	scanner := bufio.NewScanner(params.input)

	templated := strings.Contains(params.url, "{{")

	var paramsToUrl func(string) (string, error)
	if templated {
		urlTemplate, err := template.New("url").Parse(params.url)
		if err != nil {
			return fmt.Errorf("parse url template \"%s\": %w", params.url, err)
		}

		paramsToUrl = func(line string) (string, error) {
			row := splitRows(line, params.fieldSeparator)
			//TODO: explicit param or guessing for whether it's a path or not.
			result := bytes.Buffer{}
			if err := urlTemplate.Execute(&result, row); err != nil {
				return "", fmt.Errorf("building url from %q: %w", row, err)
			} else {
				return result.String(), nil
			}
		}
	} else {
		//TODO: no-escape
		appendAsQueryParameter := strings.Contains(params.url, "?")
		if appendAsQueryParameter {
			paramsToUrl = func(line string) (string, error) {
				return params.url + url.QueryEscape(line), nil
			}
		} else {
			paramsToUrl = func(line string) (string, error) {
				return params.url + url.PathEscape(line), nil
			}
		}
	}

	tracker := Tracker{
		StopOnFirstErr:            params.stopOnFirstError,
		StopOnConsecutiveErrCount: params.stopOnErrorCount,
	}

	httpClient := http.Client{
		Timeout: params.timeout,
	}

	for scanner.Scan() {
		nextLine := scanner.Text()
		if strings.TrimSpace(nextLine) == "" {
			continue
		}
		fmt.Fprint(params.output, nextLine, " ")
		urlToCall, err := paramsToUrl(nextLine)
		if err != nil {
			return err
		}

		resp, err := httpClient.Post(urlToCall, "", nil)
		if err != nil {
			if urlErr, ok := err.(*url.Error); ok {
				if urlErr.Timeout() {
					fmt.Fprintln(params.output, "ERR Timeout")
				} else {
					fmt.Fprintln(params.output, "ERR", urlErr)
				}
				if bailoutErr := tracker.Err(); bailoutErr != nil {
					return bailoutErr
				}
			} else {
				return fmt.Errorf("Unexpected error posting to %s : %w", urlToCall, err)
			}
		} else {
			defer resp.Body.Close()

			if resp.StatusCode/100 == 2 {
				fmt.Fprintf(params.output, "OK\n")
				tracker.Ok()
			} else {
				fmt.Fprintln(params.output, "ERR HTTP", resp.StatusCode)
				if bailoutErr := tracker.Err(); bailoutErr != nil {
					return bailoutErr
				}
			}
		}

	}

	return nil
}

func splitRows(input, fieldSeparators string) []string {
	if "" == fieldSeparators {
		return strings.Fields(input)
	}

	return strings.FieldsFunc(input, func(it rune) bool {
		return strings.ContainsRune(fieldSeparators, it)
	})
}

type runParams struct {
	url              string
	input            io.Reader
	output           io.Writer
	fieldSeparator   string
	stopOnErrorCount int
	stopOnFirstError bool
	timeout          time.Duration
}

func newRunParams() runParams {
	return runParams{
		input:            os.Stdin,
		output:           os.Stdout,
		stopOnErrorCount: 0,
	}
}

type Tracker struct {
	rowNo                     int
	consecutiveErrCount       int
	StopOnFirstErr            bool
	StopOnConsecutiveErrCount int
}

func (t *Tracker) Ok() {
	t.rowNo++
	t.consecutiveErrCount = 0
}

// Err returns reason to bail out if such
func (t *Tracker) Err() error {
	t.rowNo++
	t.consecutiveErrCount++
	if t.StopOnFirstErr && t.rowNo == 1 {
		return fmt.Errorf("error on first call")
	}
	if t.StopOnConsecutiveErrCount > 0 && t.consecutiveErrCount >= t.StopOnConsecutiveErrCount {
		return fmt.Errorf("%d consecutive errors", t.consecutiveErrCount)
	}
	return nil
}
