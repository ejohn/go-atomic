// +build windows

package runner

import (
	"fmt"
	"io"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type launchProc struct {
	cmd    *exec.Cmd
	job    windows.Handle
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// links to documentation on JobObject which is used to ensure child processes
// are also killed when the launcher is shut down.
// https://gist.github.com/hallazzang/76f3970bfc949831808bbebc8ca15209
// https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects
// https://devblogs.microsoft.com/oldnewthing/20131209-00/?p=2433

func createJobObject() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return windows.InvalidHandle, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {
		return windows.InvalidHandle, err
	}
	return job, nil
}

func startLauncher(launcher []string) (*launchProc, error) {
	var err error
	lp := &launchProc{}
	lp.job, err = createJobObject()
	if err != nil {
		return nil, fmt.Errorf("failed to get job object: %s", err)
	}

	if len(launcher) > 1 {
		lp.cmd = exec.Command(launcher[0], launcher[1:]...)
	} else {
		lp.cmd = exec.Command(launcher[0])
	}

	lp.stdin, lp.stdout, lp.stderr, err = getPipes(lp.cmd)
	if err != nil {
		_ = windows.CloseHandle(lp.job)
		return nil, err
	}

	err = lp.cmd.Start()
	if err != nil {
		_ = windows.CloseHandle(lp.job)
		return nil, err
	}

	// We use this struct to retrieve process handle(which is unexported)
	// from os.Process using unsafe operation.
	type process struct {
		Pid    int
		Handle uintptr
	}

	err = windows.AssignProcessToJobObject(
		lp.job,
		windows.Handle((*process)(unsafe.Pointer(lp.cmd.Process)).Handle))
	if err != nil {
		windows.CloseHandle(lp.job)
		return nil, err
	}
	return lp, nil
}

func killLauncher(lp *launchProc) {
	// closing the handle kills all the processes that were created inside the job.
	_ = windows.CloseHandle(lp.job)
}
