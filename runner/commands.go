package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/ejohn/go-atomic/art"
)

func runCommands(launcher []string, commands string, timeout *time.Duration, splitCmds bool) ([]CmdRunInfo, error) {
	if commands == "" {
		return nil, fmt.Errorf("no commands provided")
	}
	var result *CmdResult
	var cri []CmdRunInfo
	var cmdErr error

	ctx := context.Background()
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}
	if splitCmds {
		commandLines := strings.Split(commands, "\n")
		for _, command := range commandLines {
			command = strings.TrimSpace(command)
			// TODO: timeouts are applied per command instead of the whole test. change this.
			result, cmdErr = runCommand(ctx, launcher, command)
			cri = append(cri, CmdRunInfo{
				Command: command,
				Result:  result,
			})
			// bail on first error
			if cmdErr != nil {
				break
			}
		}
		return cri, cmdErr
	}

	result, cmdErr = runCommand(ctx, launcher, commands)
	cri = append(cri, CmdRunInfo{
		Command: commands,
		Result:  result,
	})
	return cri, cmdErr
}

func getPipes(cmd *exec.Cmd) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	return stdin, stdout, stderr, nil
}

// runCommand runs command using the provided launcher. It returns the combined output
// on stdout and stderr along with an exitcode
func runCommand(ctx context.Context, launcher []string, command string) (*CmdResult, error) {
	lProcess, err := startLauncher(launcher)
	if err != nil {
		return nil, err
	}
	startTime := time.Now()
	go func() {
		defer lProcess.stdin.Close()
		_, _ = io.WriteString(lProcess.stdin, command)
		// send new line to ensure executor starts executing the command
		_, _ = io.WriteString(lProcess.stdin, "\n")
	}()

	pid := lProcess.cmd.Process.Pid

	var stdoutBuf bytes.Buffer
	go func() {
		defer lProcess.stdout.Close()
		scanner := bufio.NewScanner(lProcess.stdout)
		scanner.Split(bufio.ScanBytes)
		for scanner.Scan() {
			stdoutBuf.Write(scanner.Bytes())
		}
	}()

	var stderrBuf bytes.Buffer
	go func() {
		defer lProcess.stderr.Close()
		scanner := bufio.NewScanner(lProcess.stderr)
		scanner.Split(bufio.ScanBytes)
		for scanner.Scan() {
			stderrBuf.Write(scanner.Bytes())
		}
	}()
	var exitCode int
	var cmdErr error
	cmdDone := make(chan struct{})

	go func() {
		if cmdErr = lProcess.cmd.Wait(); cmdErr != nil {
			if exitErr, ok := cmdErr.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
		}
		close(cmdDone)
	}()

	select {
	case <-ctx.Done():
		killLauncher(lProcess)
		cmdErr = fmt.Errorf("command timed out")
		exitCode = -1
	case <-cmdDone:
	}

	res := &CmdResult{
		PID:       pid,
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
		ExitCode:  exitCode,
		StartTime: startTime,
		EndTime:   time.Now(),
	}
	return res, cmdErr
}

func buildArguments(defaultArgs map[string]art.Argument, args map[string]string, atomicsFolder string) map[string]string {
	combined := make(map[string]string)
	for k, v := range defaultArgs {
		combined[k] = v.Default
	}
	for k, v := range args {
		// skip user supplied args unless they are specified in the atomic test
		if _, found := defaultArgs[k]; !found {
			continue
		}
		combined[k] = v
	}
	for k := range combined {
		arg := strings.ReplaceAll(combined[k], "$PathToAtomicsFolder", atomicsFolder)
		arg = strings.ReplaceAll(arg, "PathToAtomicsFolder", atomicsFolder)
		combined[k] = arg
	}
	return combined
}

func buildCommands(commandTemplate string, arguments map[string]string, atomicsFolder string) (string, error) {
	if commandTemplate == "" {
		return "", nil
	}
	command, err := replacePlaceholder(`#\{([a-zA-Z0-9_]+)\}`, commandTemplate, arguments)
	if err != nil {
		return "", err
	}
	command, err = replacePlaceholder(`\$\{([a-zA-Z0-9_]+)\}`, command, arguments)
	if err != nil {
		return "", err
	}
	command = strings.ReplaceAll(command, "$PathToAtomicsFolder", atomicsFolder)
	command = strings.ReplaceAll(command, "PathToAtomicsFolder", atomicsFolder)

	return strings.TrimSpace(command), err
}

func replacePlaceholder(re, commandTemplate string, arguments map[string]string) (string, error) {
	pattern1 := regexp.MustCompile(re)
	matches := pattern1.FindAllSubmatch([]byte(commandTemplate), -1)

	_command := commandTemplate
	for _, match := range matches {
		val := arguments[string(match[1])]
		if val == "" {
			return "", fmt.Errorf("did not find a replacement argument for placeholder %s", string(match[0]))
		}
		_command = strings.Replace(_command, string(match[0]), val, -1)
	}
	return _command, nil
}
