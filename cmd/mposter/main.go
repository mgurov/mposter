package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"

	"github.com/mgurov/mposter/cmd/mposter/runparams"
	"github.com/mgurov/mposter/internal/tracker"
	"github.com/mgurov/mposter/internal/urltemplate"
)

func main() {
	params, err := runparams.Parse(os.Args[0], os.Args[1:])
	if nil != err {
		os.Exit(2)
	}

	err = run(params)
	if nil != err {
		log.Fatal(err)
	}
}

func run(params runparams.RunParams) error {

	scanner := bufio.NewScanner(params.Input)

	templated := strings.Contains(params.Url, "{{")

	var paramsToUrl func(string) (string, error)
	if templated {

		f, err := urltemplate.Parse(params.Url)
		if nil != err {
			return fmt.Errorf("parse url template \"%s\": %w", params.Url, err)
		}

		paramsToUrl = func(line string) (string, error) {
			row := splitRows(line, params.FieldSeparator)
			//TODO: explicit param or guessing for whether it's a path or not.
			return f(row)
		}
	} else {
		//TODO: no-escape
		appendAsQueryParameter := strings.Contains(params.Url, "?")
		if appendAsQueryParameter {
			paramsToUrl = func(line string) (string, error) {
				return params.Url + url.QueryEscape(line), nil
			}
		} else {
			paramsToUrl = func(line string) (string, error) {
				return params.Url + url.PathEscape(line), nil
			}
		}
	}

	tracker := tracker.Tracker{
		StopOnFirstErr:            params.StopOnFirstError,
		StopOnConsecutiveErrCount: params.StopOnErrorCount,
		TickLog:                   params.LogTick,
	}

	if params.LogTick > -1 {
		tracker.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	defer func() {
		tracker.LogDone()
	}()

	httpClient := http.Client{
		Timeout: params.Timeout,
	}

	skipLines := params.Skip

	for scanner.Scan() {
		nextLine := strings.TrimSpace(scanner.Text())

		if skipLines > 0 {
			skipLines--
			continue
		}

		//TODO: this one will probably interfere with the skip lines feature.
		if nextLine == "" {
			continue
		}
		fmt.Fprint(params.Output, nextLine, " ")
		urlToCall, err := paramsToUrl(nextLine)
		if err != nil {
			return err
		}

		if params.DryRun {
			fmt.Fprintln(params.Output, params.HttpMethod, urlToCall)
			continue
		}

		req, err := http.NewRequest(params.HttpMethod, urlToCall, nil)
		if err != nil {
			return fmt.Errorf("Unexpected error creating request to %s : %w", urlToCall, err)
		}
		if params.HttpAcceptType != "" {
			req.Header.Add("Accept", params.HttpAcceptType)
		}
		if params.HttpContentType != "" {
			req.Header.Add("Content", params.HttpContentType)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if urlErr, ok := err.(*url.Error); ok {
				if urlErr.Timeout() {
					fmt.Fprintln(params.Output, "ERR Timeout")
				} else {
					fmt.Fprintln(params.Output, "ERR", urlErr)
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
				fmt.Fprintf(params.Output, "OK\n")
				tracker.Ok()
			} else {
				fmt.Fprintln(params.Output, "ERR HTTP", resp.StatusCode)
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
		return unicode.IsSpace(it) || strings.ContainsRune(fieldSeparators, it)
	})
}
