// +build windows

package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFolder = "testdata"
)

func getContextWithCancel(to *time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if to == nil {
		return ctx, nil
	}
	return context.WithTimeout(ctx, *to)
}

func TestRunCommands_Powershell(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	out, err := runCommands(ctx, launcher, "echo \"hello\"\necho \"world\"", false)
	require.NoError(t, err)
	assert.Equal(t, "hello\r\nworld\r\n", out[0].Result.Stdout)
}

func TestRunCommands_PowershellExitCodeSuccess(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	res, err := runCommands(ctx, launcher, "exit 0", false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 0, res[0].Result.ExitCode)
}

func TestRunCommands_PowershellExitCodeFail(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	ctx, _ := getContextWithCancel(nil)
	res, err := runCommands(ctx, launcher, "exit 123", false)
	require.Error(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 123, res[0].Result.ExitCode)
}

func TestRunCommands_WithTimeoutSucceed(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	timeout := time.Second * 5
	ctx, _ := getContextWithCancel(&timeout)
	out, err := runCommands(ctx, launcher, "sleep 1\necho done", false)
	require.NoError(t, err)
	require.Equal(t, 1, len(out))
	assert.Equal(t, "done\r\n", out[0].Result.Stdout)
	assert.Equal(t, 0, out[0].Result.ExitCode)
}

func TestRunCommands_WithTimeoutFail(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	timeout := time.Second * 1
	ctx, _ := getContextWithCancel(&timeout)
	out, err := runCommands(ctx, launcher, "sleep 6\necho done\n", false)
	require.Error(t, err)
	assert.Equal(t, "command timed out", err.Error())
	assert.Equal(t, "", out[0].Result.Stdout)
	assert.Equal(t, -1, out[0].Result.ExitCode)
}
