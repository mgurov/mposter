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

	paramsToUrl, err := makeParamsToUrlFun(params)
	if err != nil {
		return err
	}

	lineUrlProcessor, onProcessingDone := makeLineUrlProcessor(params)
	defer onProcessingDone() //TODO: test this is invoked

	skipLines := params.Skip

	scanner := bufio.NewScanner(params.Input)
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

		err = lineUrlProcessor(urlToCall)

		if err != nil {
			return err
		}
	}

	return nil
}

type ParamsToUrlFun func(params string) (string, error)

func makeParamsToUrlFun(params runparams.RunParams) (paramsToUrl ParamsToUrlFun, err error) {
	templated := strings.Contains(params.Url, "{{")

	if templated {

		f, err := urltemplate.Parse(params.Url)
		if nil != err {
			return nil, fmt.Errorf("parse url template \"%s\": %w", params.Url, err)
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

	return

}

type LineUrlProcessor func(urlToCall string) error
type LineUrlProcessingDone func()

func makeLineUrlProcessor(params runparams.RunParams) (LineUrlProcessor, LineUrlProcessingDone) {
	if params.DryRun {
		return func(urlToCall string) error {
			fmt.Fprintln(params.Output, params.HttpMethod, urlToCall)
			return nil
		}, func() {}
	}

	tracker := tracker.Tracker{
		StopOnFirstErr:            params.StopOnFirstError,
		StopOnConsecutiveErrCount: params.StopOnErrorCount,
		TickLog:                   params.LogTick,
	}

	if params.LogTick > -1 {
		tracker.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	httpClient := http.Client{
		Timeout: params.Timeout,
	}

	caller := HttpCaller{
		Tracker:    &tracker,
		HttpClient: &httpClient,
		Params:     params,
	}

	return caller.Call, tracker.LogDone
}

func splitRows(input, fieldSeparators string) []string {
	if "" == fieldSeparators {
		return strings.Fields(input)
	}

	return strings.FieldsFunc(input, func(it rune) bool {
		return unicode.IsSpace(it) || strings.ContainsRune(fieldSeparators, it)
	})
}

type HttpCaller struct {
	//ParamsToUrl func(string) (string, error)
	Tracker    *tracker.Tracker
	HttpClient *http.Client
	Params     runparams.RunParams
}

func (c HttpCaller) Call(urlToCall string) error {
	req, err := http.NewRequest(c.Params.HttpMethod, urlToCall, nil)
	if err != nil {
		return fmt.Errorf("Unexpected error creating request to %s : %w", urlToCall, err)
	}
	if c.Params.HttpAcceptType != "" {
		req.Header.Add("Accept", c.Params.HttpAcceptType)
	}
	if c.Params.HttpContentType != "" {
		req.Header.Add("Content", c.Params.HttpContentType)
	}

	resp, err := c.HttpClient.Do(req)

	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			if urlErr.Timeout() {
				fmt.Fprintln(c.Params.Output, "ERR Timeout")
			} else {
				fmt.Fprintln(c.Params.Output, "ERR", urlErr)
			}

			if bailoutErr := c.Tracker.Err(); bailoutErr != nil {
				return bailoutErr
			} else {
				return nil //swallow the error
			}

		} else {
			return fmt.Errorf("Unexpected error posting to %s : %w", urlToCall, err)
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		fmt.Fprintf(c.Params.Output, "OK\n")
		c.Tracker.Ok()
	} else {
		fmt.Fprintln(c.Params.Output, "ERR HTTP", resp.StatusCode)
		if bailoutErr := c.Tracker.Err(); bailoutErr != nil {
			return bailoutErr
		}
	}
	return nil
}
