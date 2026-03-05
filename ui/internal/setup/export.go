package setup

import (
	"os/exec"
	"sync"
)

func runCmd(workDir string, env []string, log LogFunc, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workDir
	if env != nil {
		cmd.Env = env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go streamLines(stdout, log, &wg)
	go streamLines(stderr, log, &wg)
	wg.Wait()

	return cmd.Wait()
}
