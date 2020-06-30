package runner

import "time"

// BuiltTest represents an atomic test after its commands have been substituted
// with the supplied input arguments. A built test is what will be run by the runner.
type BuiltTest struct {
	TechniqueID        string
	TestName           string
	TestGUID           string
	Platform           string
	Executor           string
	Launcher           []string
	Arguments          map[string]string
	DependencyInfo     *DependencyInfo
	AtomicTestCommands string
	CleanupCommands    string
}

// DependencyInfo represents all the built dependencies needed for the test to run
// and specifies the executor to use.
type DependencyInfo struct {
	Executor     string
	Launcher     []string
	Dependencies []BuiltDependency

	supportedExecutor bool
}

// BuiltDependency represents one built dependency for a test case.
type BuiltDependency struct {
	PreReqCmds    string
	GetPreReqCmds string
}

// TestRunInfo represents the details of an atomic test and the results of running it.
type TestRunInfo struct {
	TechniqueID    string
	TestName       string
	TestGUID       string
	Platform       string
	Executor       string
	Launcher       []string
	Arguments      map[string]string
	DependencyInfo *DependencyRunInfo
	AtomicTest     []CmdRunInfo
	Cleanup        []CmdRunInfo
}

// CmdRunInfo represents one set of commands to run and their results.
type CmdRunInfo struct {
	Command string
	Result  *CmdResult
}

// DependencyRunInfo represents all the dependencies that were ran for a test case and their results.
type DependencyRunInfo struct {
	Launcher     string
	Dependencies []DependencyRunResults
}

// DependencyRunResults represents the commands and results of running one dependency.
type DependencyRunResults struct {
	PreReq    []CmdRunInfo
	GetPreReq []CmdRunInfo
}

// CmdResult represents the results of a command execution.
type CmdResult struct {
	PID       int
	Stdout    string
	Stderr    string
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}

// FilterConfig represents options to filter techniques.
type FilterConfig struct {
	Platform      string
	Techniques    []string
	IncludeManual bool
}

// TestRunConfig represents the options to control the execution of a test.
type TestRunConfig struct {
	// EnableAll is equivalent to setting all the individual enable flags to true.
	// However there is a deference in the behavior around how errors are returned
	// when commands are empty.
	EnableAll bool

	EnableCheckPreReq bool
	EnableTest        bool
	EnableCleanup     bool
	EnableDependency  bool

	SplitCmdsByNewline bool
}
