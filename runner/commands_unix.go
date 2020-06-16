// +build !windows

package runner

import (
	"io"
	"os/exec"
	"syscall"
)

type launchProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func startLauncher(launcher []string) (*launchProc, error) {
	lp := &launchProc{}
	if len(launcher) > 1 {
		lp.cmd = exec.Command(launcher[0], launcher[1:]...)
	} else {
		lp.cmd = exec.Command(launcher[0])
	}
	// create new process group for launcher
	lp.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var err error
	lp.stdin, lp.stdout, lp.stderr, err = getPipes(lp.cmd)
	if err != nil {
		return nil, err
	}

	err = lp.cmd.Start()
	if err != nil {
		return nil, err
	}

	return lp, nil
}

func killLauncher(lp *launchProc) {
	// kill all processes in the process group
	_ = syscall.Kill(-lp.cmd.Process.Pid, syscall.SIGKILL)
}
