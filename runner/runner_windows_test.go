// +build windows

package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFolder = "testdata"
)

func TestRunCommands_Powershell(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	out, err := runCommands(launcher, "echo \"hello\"\necho \"world\"", nil, false)
	require.NoError(t, err)
	assert.Equal(t, "hello\r\nworld\r\n", out[0].Result.Stdout)
}

func TestRunCommands_PowershellExitCodeSuccess(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	res, err := runCommands(launcher, "exit 0", nil, false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 0, res[0].Result.ExitCode)
}

func TestRunCommands_PowershellExitCodeFail(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	res, err := runCommands(launcher, "exit 123", nil, false)
	require.Error(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 123, res[0].Result.ExitCode)
}

func TestRunCommands_WithTimeoutSucceed(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	timeout := time.Second * 5
	out, err := runCommands(launcher, "sleep 1\necho done", &timeout, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(out))
	assert.Equal(t, "done\r\n", out[0].Result.Stdout)
	assert.Equal(t, 0, out[0].Result.ExitCode)
}

func TestRunCommands_WithTimeoutFail(t *testing.T) {
	launcher, err := getLauncher("powershell")
	require.NoError(t, err)
	timeout := time.Second * 1
	out, err := runCommands(launcher, "sleep 6\necho done\n", &timeout, false)
	require.Error(t, err)
	assert.Equal(t, "command timed out", err.Error())
	assert.Equal(t, "", out[0].Result.Stdout)
	assert.Equal(t, -1, out[0].Result.ExitCode)
}
