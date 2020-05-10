package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mgurov/mposter/internal/tracker"
	"github.com/mgurov/mposter/internal/urltemplate"
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
	commandLine.BoolVar(&result.dryRun, "dry-run", false, "prints the http calls instead of executing them if true")
	commandLine.IntVar(&result.stopOnErrorCount, "stop-on-err-count", result.stopOnErrorCount, "Stop on consequent error results")
	commandLine.BoolVar(&result.stopOnFirstError, "stop-on-first-err", true, "stop on very first error at once, disregarding the stop-on-err-count setting")
	commandLine.DurationVar(&result.timeout, "timeout", 0, "http timeout, 0 (default) meaning no timeout")
	commandLine.IntVar(&result.logTick, "tick", 1000, "How often to log the summary status to stderr. 0 to only log the final statistics. -1 to disable the logging whatsoever.")
	commandLine.BoolVar(&result.logFirstErrStatus, "log-first-err-stats", true, "log status to stderr upon first error encountered")
	commandLine.StringVar(&result.httpContentType, "http-content-type", "", "specify the value for the Content http request header")
	commandLine.StringVar(&result.httpAcceptType, "http-accept-type", "*/*", "specify the value for the Accept http request header")
	commandLine.StringVar(&result.httpMethod, "http-method", "POST", "http method")
	commandLine.IntVar(&result.skip, "skip", 0, "skip first lines, e.g. header or continue")

	return result, commandLine.Parse(os.Args[1:])
}

func run(params runParams) error {

	scanner := bufio.NewScanner(params.input)

	templated := strings.Contains(params.url, "{{")

	var paramsToUrl func(string) (string, error)
	if templated {

		f, err := urltemplate.Parse(params.url)
		if nil != err {
			return fmt.Errorf("parse url template \"%s\": %w", params.url, err)
		}

		paramsToUrl = func(line string) (string, error) {
			row := splitRows(line, params.fieldSeparator)
			//TODO: explicit param or guessing for whether it's a path or not.
			return f(row)
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

	tracker := tracker.Tracker{
		StopOnFirstErr:            params.stopOnFirstError,
		StopOnConsecutiveErrCount: params.stopOnErrorCount,
		TickLog:                   params.logTick,
	}

	if params.logTick > -1 {
		tracker.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	defer func() {
		tracker.LogDone()
	}()

	httpClient := http.Client{
		Timeout: params.timeout,
	}

	skipLines := params.skip

	for scanner.Scan() {
		nextLine := scanner.Text()

		if skipLines > 0 {
			skipLines--
			continue
		}

		//TODO: this one will probably interfere with the skip lines feature.
		if strings.TrimSpace(nextLine) == "" {
			continue
		}
		fmt.Fprint(params.output, nextLine, " ")
		urlToCall, err := paramsToUrl(nextLine)
		if err != nil {
			return err
		}

		if params.dryRun {
			fmt.Fprintln(params.output, "POST", urlToCall)
			continue
		}

		req, err := http.NewRequest(params.httpMethod, urlToCall, nil)
		if err != nil {
			return fmt.Errorf("Unexpected error creating request to %s : %w", urlToCall, err)
		}
		if params.httpAcceptType != "" {
			req.Header.Add("Accept", params.httpAcceptType)
		}
		if params.httpContentType != "" {
			req.Header.Add("Content", params.httpContentType)
		}

		resp, err := httpClient.Do(req)
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
	url               string
	input             io.Reader
	output            io.Writer
	fieldSeparator    string
	stopOnErrorCount  int
	stopOnFirstError  bool
	logTick           int
	logFirstErrStatus bool
	timeout           time.Duration
	dryRun            bool
	skip              int
	httpAcceptType    string
	httpContentType   string
	httpMethod        string
}

func newRunParams() runParams {
	return runParams{
		input:            os.Stdin,
		output:           os.Stdout,
		stopOnErrorCount: 0,
	}
}
