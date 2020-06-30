package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/ejohn/go-atomic/art"
	"github.com/ejohn/go-atomic/runner"
)

var logger *log.Logger

func main() {
	opts, err := processFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(run(opts))
}

func validateOptions(f *options) error {
	if f.atomicsFolder == "" {
		return fmt.Errorf("path to atomics folder is required")
	}
	// cannot specify both at both number and test name at the same time.
	if f.number != "" && f.testName != "" {
		return fmt.Errorf("cannot specify both -num and -name at the same time")
	}

	// test name and number are not valid without a technique being set
	if f.techniqueID == "" {
		if f.testName != "" || f.number != "" {
			return fmt.Errorf("a techniqueID must be specified in order to select using test name or number")
		}
	}

	if f.runAll && (f.runExecutor || f.runCheckPreReq || f.runDependency || f.runCleanup) {
		return fmt.Errorf("-run and options like -cleanup or -dependency cannot be specified at the same time")
	}

	if f.isRun && f.dryRun {
		return fmt.Errorf("-dry-run and options like -test or -dependency cannot be specified at the same time")
	}

	return nil
}

func run(f *options) int {
	err := validateOptions(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		flag.Usage()
		return 1
	}

	// process test arguments
	var testArguments map[string]string
	if testArguments, err = processArguments(f.arguments); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	ar := &runner.Runner{
		AtomicsFolder: f.atomicsFolder,
	}
	// set debug logger
	if f.debug {
		ar.Logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	err = ar.LoadTechniques()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	var techniqueIDs []string
	if f.techniqueID != "" {
		techniqueIDs = strings.Split(f.techniqueID, ",")
	}
	var guids []string
	if f.guid != "" {
		guids = strings.Split(f.guid, ",")
	}

	if len(guids) > 0 && len(techniqueIDs) > 0 {
		fmt.Println(guids, len(guids))
		fmt.Fprintf(os.Stderr, "specify either a technique or test guid at a time")
		return 1
	}

	if len(guids) > 0 {
		for _, guid := range guids {
			handleGUID(ar, testArguments, guid, f)
		}
		return 0
	}

	// single technique with an number
	if len(techniqueIDs) == 1 && f.number != "" {
		return handleTechNumber(ar, testArguments, f)
	}

	// single technique with a test name
	if len(techniqueIDs) == 1 && f.testName != "" {
		return handleTechName(ar, testArguments, f)
	}

	// single tests have been handled. if number is set at this stage, the options are set incorrectly.
	if f.number != "" {
		fmt.Fprintf(os.Stderr, "-num is supported only when used with a single technique\n"+
			"examples:\n\t-tech T1002 -num 0")
		return 1
	}

	// single tests have been handled. if name is set at this stage, the options are set incorrectly.
	if f.testName != "" {
		fmt.Fprintf(os.Stderr, "-name is supported only when used with a single technique\n"+
			"examples:\n\t-tech T1002 -name \"Data Compressed - nix - zip\"\n")
		return 1
	}
	return handleFilterTests(techniqueIDs, ar, testArguments, f)
}

func handleFilterTests(techniqueIDs []string, ar *runner.Runner, testArguments map[string]string, options *options) int {
	fc := &runner.FilterConfig{
		Platform:      "",
		Techniques:    techniqueIDs,
		IncludeManual: true,
	}
	filtered := ar.Filter(fc)
	if len(filtered) == 0 {
		fmt.Fprintf(os.Stderr, "no tests found matching criteria os: %s, techniques: %v\n",
			runtime.GOOS, techniqueIDs)
		return 1
	}
	for _, tech := range filtered {
		runTechnique(ar, tech, testArguments, options)
	}
	return 0
}

func handleGUID(ar *runner.Runner, testArguments map[string]string, guid string, options *options) int {
	at, err := ar.GetTestByGUID(strings.TrimSpace(guid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	return runTest(ar, at, testArguments, options)
}

func handleTechName(ar *runner.Runner, testArguments map[string]string, options *options) int {
	at, err := ar.GetTestByIDAndName(options.techniqueID, strings.TrimSpace(options.testName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	return runTest(ar, at, testArguments, options)
}

func handleTechNumber(ar *runner.Runner, testArguments map[string]string, options *options) int {
	testNumber, err := strconv.Atoi(strings.TrimSpace(options.number))
	if err != nil || testNumber == 0 {
		fmt.Fprintf(os.Stderr, "invalid test number, valid test numbers are 1-N")
		return 1
	}
	at, err := ar.GetTestByIDAndIndex(options.techniqueID, testNumber-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	return runTest(ar, at, testArguments, options)
}

type args []string

func (i *args) String() string {
	return strings.Join(*i, " ")
}
func (i *args) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type options struct {
	atomicsFolder string
	techniqueID   string
	number        string
	testName      string
	guid          string

	timeout string

	dryRun bool

	runExecutor    bool
	runCheckPreReq bool
	runCleanup     bool
	runDependency  bool
	runAll         bool

	isRun bool
	debug bool

	arguments args

	parsedTimeout *time.Duration
}

func processFlags() (*options, error) {
	opts := options{}
	flag.StringVar(&opts.atomicsFolder, "path", "", "path to atomics folder")

	flag.StringVar(&opts.techniqueID, "tech", "", "list of technique id's [ex T1002,T1003]")
	flag.StringVar(&opts.number, "num", "", "test case number [1-N]")
	flag.StringVar(&opts.testName, "name", "", "name of the test to run")
	flag.StringVar(&opts.guid, "guid", "", "test case guids separated by comma")

	flag.StringVar(&opts.timeout, "timeout", "", "timeout for commands [ex 1s, 2m]")

	flag.BoolVar(&opts.dryRun, "dry-run", false, "build test and display what will be executed "+
		"when the test is run")

	flag.BoolVar(&opts.runAll, "run", false, "run dependencies, test commands and cleanup for all tests selected")

	flag.BoolVar(&opts.runCheckPreReq, "prereq", false, "check if prerequisites for test are met")
	flag.BoolVar(&opts.runCleanup, "cleanup", false, "run only cleanup")
	flag.BoolVar(&opts.runExecutor, "test", false, "run only the test, disables dependencies and cleanup")
	flag.BoolVar(&opts.runDependency, "dependency", false, "check prerequisites and get "+
		"them if needed")

	flag.BoolVar(&opts.debug, "debug", false, "show debug logs")
	flag.Var(&opts.arguments, "arg", "pass argument to test [ex foo=bar], "+
		"set multiple times for different arguments")

	flag.Parse()

	if opts.runAll || opts.runExecutor || opts.runCheckPreReq || opts.runDependency || opts.runCleanup {
		opts.isRun = true
	}

	if opts.timeout != "" {
		pt, err := time.ParseDuration(opts.timeout)
		if err != nil {
			return nil, fmt.Errorf("timeout is not valid")
		}
		opts.parsedTimeout = &pt
	}

	return &opts, nil
}

func processArguments(arguments args) (map[string]string, error) {
	var testArguments map[string]string
	if len(arguments) > 0 {
		testArguments = make(map[string]string)
		for _, arg := range arguments {
			index := strings.Index(arg, "=")
			if index <= 0 {
				return nil, fmt.Errorf("argument %s is not properly formated, use key=val format", arg)
			}
			testArguments[arg[0:index]] = arg[index+1:]
		}
	}
	return testArguments, nil
}

func getRC(f *options) *runner.TestRunConfig {
	rc := runner.TestRunConfig{
		EnableCheckPreReq:  f.runCheckPreReq,
		EnableTest:         f.runExecutor,
		EnableCleanup:      f.runCleanup,
		EnableDependency:   f.runDependency,
		SplitCmdsByNewline: false,
	}
	if f.runAll {
		rc.EnableAll = true
	}
	return &rc
}

func runTest(ar *runner.Runner, at *art.Test, testArguments map[string]string, options *options) int {
	if options.dryRun {
		br, err := ar.BuildTest(at, testArguments)
		displayBuiltTestInfo(br, err)
		return 0
	}
	if options.runCleanup && at.Executor.CleanupCommand == "" {
		fmt.Fprintf(os.Stderr, "no cleanup command for test %s:%s", at.TechniqueID, at.Name)
		return 1
	}
	if (options.runDependency || options.runCheckPreReq) && len(at.Dependencies) == 0 {
		fmt.Fprintf(os.Stderr, "no dependencies for test %s:%s", at.TechniqueID, at.Name)
		return 1
	}

	rc := getRC(options)
	var cancel context.CancelFunc
	ctx := context.Background()
	if options.parsedTimeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *options.parsedTimeout)
		defer cancel()
	}

	if options.isRun {
		// error is ignored here since the error message is also part of the result struct
		atr, err := ar.RunTest(ctx, at, testArguments, rc)
		if options.debug {
			logger.Printf("%s\n", err)
		}
		displayTestResult(atr, err)
		return 0
	}
	displayTestInfo(at)
	return 0
}

func runTechnique(ar *runner.Runner, tech *art.Technique, testArguments map[string]string, options *options) {
	for _, test := range tech.AtomicTests {
		// ignore return codes when multiple tests are run
		runTest(ar, test, testArguments, options)
	}
}

func displayTestInfo(atomicTest *art.Test) {
	dumpJSON(atomicTest)
}

func displayBuiltTestInfo(bt *runner.BuiltTest, err error) {
	var errString string
	if err != nil {
		errString = err.Error()
	}
	res := struct {
		*runner.BuiltTest
		Error string
	}{
		bt,
		errString,
	}
	dumpJSON(res)
}

func getErrorMessages(err error) []string {
	if err == nil {
		return nil
	}
	var errMsgs []string
	if merrs, ok := err.(*multierror.Error); ok {
		for _, merr := range merrs.Errors {
			errMsgs = append(errMsgs, merr.Error())
		}
		return errMsgs
	}
	return []string{err.Error()}
}
func displayTestResult(result *runner.TestRunInfo, err error) {
	res := struct {
		*runner.TestRunInfo
		Error []string
	}{
		result,
		getErrorMessages(err),
	}
	dumpJSON(res)
}

func dumpJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	err := enc.Encode(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
}
