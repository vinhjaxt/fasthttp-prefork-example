package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
)

var preforkChildEnvVariable = `PREFORK_IS_CHILD`
var PreforkErrOverRecovery = errors.New("exceeding the value of RecoverThreshold")

func PreforkForkChild(f []*os.File) (*exec.Cmd, error) {
	/* #nosec G204 */
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), preforkChildEnvVariable+"=1")
	cmd.ExtraFiles = f
	err := cmd.Start()
	return cmd, err
}

func Prefork(f []*os.File, recoverThreshold int) (err error) {
	if recoverThreshold < 1 {
		recoverThreshold = runtime.GOMAXPROCS(0) / 2
	}

	type procSig struct {
		pid int
		err error
	}

	goMaxProcs := runtime.GOMAXPROCS(0)
	sigCh := make(chan procSig, goMaxProcs)
	childProcs := make(map[int]*exec.Cmd)

	defer func() {
		for _, proc := range childProcs {
			_ = proc.Process.Kill()
		}
	}()

	for i := 0; i < goMaxProcs; i++ {
		var cmd *exec.Cmd
		if cmd, err = PreforkForkChild(f); err != nil {
			log.Printf("failed to start a child prefork process, error: %v\n", err)
			return
		}

		childProcs[cmd.Process.Pid] = cmd
		go func() {
			sigCh <- procSig{cmd.Process.Pid, cmd.Wait()}
		}()
	}

	var exitedProcs int
	for sig := range sigCh {
		delete(childProcs, sig.pid)

		log.Printf("one of the child prefork processes exited with "+
			"error: %v", sig.err)

		exitedProcs++
		if exitedProcs > recoverThreshold {
			log.Printf("child prefork processes exit too many times, "+
				"which exceeds the value of RecoverThreshold(%d), "+
				"exiting the master process.\n", exitedProcs)
			err = PreforkErrOverRecovery
			break
		}

		var cmd *exec.Cmd
		if cmd, err = PreforkForkChild(f); err != nil {
			break
		}
		childProcs[cmd.Process.Pid] = cmd
		go func() {
			sigCh <- procSig{cmd.Process.Pid, cmd.Wait()}
		}()
	}

	return
}

func PreforkGetListenerFd() (net.Listener, error) {
	return net.FileListener(os.NewFile(3, ""))
}

// PreforkIsChild checks if the current thread/process is a child.
func PreforkIsChild() bool {
	return os.Getenv(preforkChildEnvVariable) == `1`
}
