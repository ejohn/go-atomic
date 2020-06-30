// +build darwin linux

package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejohn/go-atomic/art"
)

const (
	testFolder = "testdata"
)

func getDefaultRC() *TestRunConfig {
	return &TestRunConfig{
		EnableAll:          true,
		SplitCmdsByNewline: false,
	}
}

func getContextWithCancel(to *time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if to == nil {
		return ctx, nil
	}
	return context.WithTimeout(ctx, *to)
}

func TestRunCommands(t *testing.T) {
	launcher, err := getLauncher("command_prompt")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	out, err := runCommands(ctx, launcher, "echo \"hello\"\necho \"world\"", false)
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", out[0].Result.Stdout)
}

func TestRunCommands_WithTimeoutSucceed(t *testing.T) {
	launcher, err := getLauncher("command_prompt")
	require.NoError(t, err)
	timeout := time.Second * 5
	ctx, _ := getContextWithCancel(&timeout)
	out, err := runCommands(ctx, launcher, "sleep 1\necho done", false)
	require.NoError(t, err)
	require.Equal(t, 1, len(out))
	assert.Equal(t, "done\n", out[0].Result.Stdout)
	assert.Equal(t, 0, out[0].Result.ExitCode)
}

func TestRunCommands_WithTimeoutFail(t *testing.T) {
	launcher, err := getLauncher("bash")
	require.NoError(t, err)
	timeout := time.Second * 1
	ctx, _ := getContextWithCancel(&timeout)
	out, err := runCommands(ctx, launcher, "sleep 6\necho done\n", false)
	require.Error(t, err)
	assert.Equal(t, "command timed out", err.Error())
	assert.Equal(t, "", out[0].Result.Stdout)
	assert.Equal(t, -1, out[0].Result.ExitCode)
}

func TestRunTest(t *testing.T) {
	atomicTest, args := getMockTest()
	ar := Runner{}
	ctx, _ := getContextWithCancel(nil)
	out, err := ar.RunTest(ctx, atomicTest, args, getDefaultRC())
	require.Nil(t, err)
	assert.Equal(t, "command-user\n", out.AtomicTest[0].Result.Stdout)
	assert.Equal(t, "cleanup-user\n", out.Cleanup[0].Result.Stdout)
	require.NotNil(t, out.DependencyInfo)
	assert.Equal(t, 2, len(out.DependencyInfo.Dependencies))

	// results of first dependency check
	assert.Equal(t, 123, out.DependencyInfo.Dependencies[0].PreReq[0].Result.ExitCode)
	assert.Equal(t, "prereq-user\n", out.DependencyInfo.Dependencies[0].PreReq[0].Result.Stdout)
	assert.Equal(t, 0, out.DependencyInfo.Dependencies[0].GetPreReq[0].Result.ExitCode)
	assert.Equal(t, "getprereq-default\n", out.DependencyInfo.Dependencies[0].GetPreReq[0].Result.Stdout)

	// results of second dependency check.
	// getprereq should not run since exit code for prereq should be 0.
	assert.Equal(t, 0, out.DependencyInfo.Dependencies[1].PreReq[0].Result.ExitCode)
	assert.Equal(t, "prereq-user\n", out.DependencyInfo.Dependencies[1].PreReq[0].Result.Stdout)
	assert.Nil(t, out.DependencyInfo.Dependencies[1].GetPreReq)
}

func TestRunTest_UnSupportedExecutor(t *testing.T) {
	atomicTest := art.Test{
		TechniqueID:        "T9999",
		Name:               "Test",
		Description:        "Test",
		SupportedPlatforms: []string{getCurrentPlatform()},
		InputArguments:     nil,
		Executor: art.Executor{
			// supported executors are [sh, bash, powershell, command_prompt].
			// although sh is a supported executor, lookup for "/bin/sh" fails
			// and execution falls back to line by line.
			Name:              "/bin/sh",
			ElevationRequired: false,
			Command:           "echo test\necho test\nexit 123\n",
		},
	}
	ar := Runner{}
	ctx, _ := getContextWithCancel(nil)
	out, err := ar.RunTest(ctx, &atomicTest, nil, getDefaultRC())
	require.Error(t, err)
	require.Equal(t, 1, len(out.AtomicTest))
	assert.Equal(t, "test\ntest\n", out.AtomicTest[0].Result.Stdout)
	assert.Equal(t, 123, out.AtomicTest[0].Result.ExitCode)
}

func TestRunTest_RunConfigSplitLines(t *testing.T) {
	atomicTest := art.Test{
		TechniqueID:        "T9999",
		Name:               "Test",
		Description:        "Test",
		SupportedPlatforms: []string{getCurrentPlatform()},
		InputArguments:     nil,
		Executor: art.Executor{
			// supported executors are [sh, bash, powershell, command_prompt].
			// although sh is a supported executor, lookup for "/bin/sh" fails
			// and execution falls back to line by line.
			Name:              "/bin/sh",
			ElevationRequired: false,
			Command:           "echo test1\necho test2\nexit 123\n",
		},
	}
	ar := Runner{}
	rc := &TestRunConfig{
		EnableTest:         true,
		SplitCmdsByNewline: true,
	}
	ctx, _ := getContextWithCancel(nil)
	out, err := ar.RunTest(ctx, &atomicTest, nil, rc)
	require.Error(t, err)
	require.Equal(t, 3, len(out.AtomicTest))
	assert.Equal(t, "test1\n", out.AtomicTest[0].Result.Stdout)
	assert.Equal(t, "test2\n", out.AtomicTest[1].Result.Stdout)
	assert.Equal(t, 123, out.AtomicTest[2].Result.ExitCode)
}

func TestRunTest_RunConfigCleanup(t *testing.T) {
	atomicTest := art.Test{
		TechniqueID:        "T9999",
		Name:               "Test",
		Description:        "Test",
		SupportedPlatforms: []string{getCurrentPlatform()},
		InputArguments:     nil,
		Executor: art.Executor{
			// supported executors are [sh, bash, powershell, command_prompt].
			// although sh is a supported executor, lookup for "/bin/sh" fails
			// and execution falls back to line by line.
			Name:              "/bin/sh",
			ElevationRequired: false,
			Command:           "echo command\n",
			CleanupCommand:    "echo cleanup\n",
		},
	}
	ar := Runner{}
	rc := &TestRunConfig{
		EnableCleanup:      true,
		SplitCmdsByNewline: false,
	}
	ctx, _ := getContextWithCancel(nil)
	out, err := ar.RunTest(ctx, &atomicTest, nil, rc)
	require.NoError(t, err)
	require.Equal(t, 0, len(out.AtomicTest))
	require.Nil(t, out.DependencyInfo)
	require.Equal(t, 1, len(out.Cleanup))
	assert.Equal(t, "cleanup\n", out.Cleanup[0].Result.Stdout)
	assert.Equal(t, 0, out.Cleanup[0].Result.ExitCode)
}

func TestRunTest_RunConfigDependency(t *testing.T) {
	atomicTest := art.Test{
		TechniqueID:            "T9999",
		Name:                   "Test",
		Description:            "Test",
		SupportedPlatforms:     []string{getCurrentPlatform()},
		InputArguments:         nil,
		DependencyExecutorName: "bash",
		Dependencies: []art.Dependency{
			{
				Description:      "test dependency 1",
				PrereqCommand:    "echo prereq1\nexit 1\n",
				GetPrereqCommand: "echo getprereq1",
			},
			{
				Description:      "test dependency 2",
				PrereqCommand:    "echo prereq2",
				GetPrereqCommand: "echo getprereq2",
			},
		},
		Executor: art.Executor{
			// supported executors are [sh, bash, powershell, command_prompt].
			// although sh is a supported executor, lookup for "/bin/sh" fails
			// and execution falls back to line by line.
			Name:              "/bin/sh",
			ElevationRequired: false,
			Command:           "echo command\n",
			CleanupCommand:    "echo cleanup\n",
		},
	}
	ar := Runner{}
	rc := &TestRunConfig{
		EnableCheckPreReq:  true,
		SplitCmdsByNewline: false,
	}
	ctx, _ := getContextWithCancel(nil)
	// test with ony prereq enabled
	out, err := ar.RunTest(ctx, &atomicTest, nil, rc)
	require.Nil(t, err)
	require.Equal(t, 0, len(out.AtomicTest))
	require.Equal(t, 0, len(out.Cleanup))
	require.NotNil(t, out.DependencyInfo)
	assert.Equal(t, "prereq1\n", out.DependencyInfo.Dependencies[0].PreReq[0].Result.Stdout)
	assert.Equal(t, "prereq2\n", out.DependencyInfo.Dependencies[1].PreReq[0].Result.Stdout)
	assert.Equal(t, 0, len(out.DependencyInfo.Dependencies[0].GetPreReq))
	assert.Equal(t, 0, len(out.DependencyInfo.Dependencies[1].GetPreReq))

	// test with both prereq and getprereq enabled
	rc.EnableDependency = true
	ctx, _ = getContextWithCancel(nil)
	out, err = ar.RunTest(ctx, &atomicTest, nil, rc)
	require.Nil(t, err)
	require.Equal(t, 0, len(out.AtomicTest))
	require.Equal(t, 0, len(out.Cleanup))
	require.NotNil(t, out.DependencyInfo)
	assert.Equal(t, "prereq1\n", out.DependencyInfo.Dependencies[0].PreReq[0].Result.Stdout)
	assert.Equal(t, "prereq2\n", out.DependencyInfo.Dependencies[1].PreReq[0].Result.Stdout)
	assert.Equal(t, 1, len(out.DependencyInfo.Dependencies[0].GetPreReq))
	assert.Equal(t, "getprereq1\n", out.DependencyInfo.Dependencies[0].GetPreReq[0].Result.Stdout)
}

func TestRunCommands_ExitCodeSuccess(t *testing.T) {
	launcher, err := getLauncher("sh")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	res, err := runCommands(ctx, launcher, "exit 0", false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 0, res[0].Result.ExitCode)
}

func TestRunCommands_ExitCodeFail(t *testing.T) {
	launcher, err := getLauncher("sh")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	res, err := runCommands(ctx, launcher, "exit 123", false)
	require.Error(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 123, res[0].Result.ExitCode)
}
