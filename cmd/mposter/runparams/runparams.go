package runparams

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type RunParams struct {
	Input  io.Reader //TODO: test
	Output io.Writer //TODO: test

	Url             string
	HttpAcceptType  string
	HttpContentType string
	HttpMethod      string
	Timeout         time.Duration

	FieldSeparator    string
	Skip              int
	DryRun            bool
	LogFirstErrStatus bool
	LogTick           int
	StopOnErrorCount  int
	StopOnFirstError  bool
}

func NewRunParams() RunParams {
	return RunParams{
		Input:             os.Stdin,
		Output:            os.Stdout,
		StopOnErrorCount:  0,
		StopOnFirstError:  true,
		LogTick:           1000,
		LogFirstErrStatus: true,
		HttpAcceptType:    "*/*",
		HttpMethod:        "POST",
	}
}

// Parse configures standard go flag with RunParams and parsses the provided command line args
// Two passes are performed to allow mixing flagless Url with other flags, e.g. `mposter --separator=x http://url/ --dry-run`
// The usage and error messages are printed to stderr if needed, so the caller doesn't have to perform those actions upon receiving non-nil error
func Parse(appname string, args []string) (RunParams, error) {

	//TODO: support reading to and writing from a file
	result := NewRunParams()

	commandLine := flag.NewFlagSet(appname, flag.ContinueOnError)
	commandLine.Usage = func() {
		fmt.Fprint(os.Stderr, "\nmposter [params] url\n\nWhere the params are:\n\n")
		commandLine.PrintDefaults()
	}
	configureFlagSet(commandLine, &result)

	err := commandLine.Parse(args)

	if err != nil {
		return result, err
	}

	customErrReporting := func(err error) error {
		if flag.ErrHelp != err {
			fmt.Fprintln(commandLine.Output(), err.Error())
		}
		commandLine.Usage()
		return err
	}

	if len(commandLine.Args()) == 0 {
		return result, customErrReporting(fmt.Errorf("url not provided"))
	}

	result.Url = commandLine.Arg(0)

	// Second pass to allow arguments passed after URL
	if len(commandLine.Args()) > 1 {
		subCommandLine := flag.NewFlagSet(appname, flag.ContinueOnError)
		// since defaults would be affected by the first pass, suppress flag's output and take care of it ourself via customErrReporting
		subCommandLine.SetOutput(ioutil.Discard)
		configureFlagSet(subCommandLine, &result)
		if err = subCommandLine.Parse(commandLine.Args()[1:]); err != nil {
			return result, customErrReporting(err)
		}
		if len(subCommandLine.Args()) != 0 {
			return result, customErrReporting(fmt.Errorf("multiple urls provided"))
		}
	}

	return result, nil
}

func configureFlagSet(flagSet *flag.FlagSet, params *RunParams) {
	flagSet.StringVar(&params.FieldSeparator, "separator", "", "row field separator. White space if not specified.")
	//TODO: document
	flagSet.BoolVar(&params.DryRun, "dry-run", params.DryRun, "prints the http calls instead of executing them if true")
	flagSet.IntVar(&params.StopOnErrorCount, "stop-on-err-count", params.StopOnErrorCount, "Stop on consequent error results")
	flagSet.BoolVar(&params.StopOnFirstError, "stop-on-first-err", params.StopOnFirstError, "stop on very first error at once, disregarding the stop-on-err-count setting")
	flagSet.DurationVar(&params.Timeout, "timeout", params.Timeout, "http timeout, 0 (default) meaning no timeout")
	flagSet.IntVar(&params.LogTick, "tick", params.LogTick, "How often to log the summary status to stderr. 0 to only log the final statistics. -1 to disable the logging whatsoever.")
	flagSet.BoolVar(&params.LogFirstErrStatus, "log-first-err-stats", params.LogFirstErrStatus, "log status to stderr upon first error encountered")
	flagSet.StringVar(&params.HttpContentType, "http-content-type", params.HttpContentType, "specify the value for the Content http request header")
	flagSet.StringVar(&params.HttpAcceptType, "http-accept-type", params.HttpAcceptType, "specify the value for the Accept http request header")
	flagSet.StringVar(&params.HttpMethod, "http-method", params.HttpMethod, "http method")
	flagSet.IntVar(&params.Skip, "skip", params.Skip, "skip first lines, e.g. header or continue")
}
