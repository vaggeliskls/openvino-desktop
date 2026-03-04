package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// PrepareExport installs ML requirements into the OVMS bundled Python using pip.
// installDir is the base directory (e.g. C:\Users\user\openvino-desk).
// requirementsURL is the URL to the requirements.txt file.
// OVMS must already be installed at installDir/ovms before calling this.
func PrepareExport(installDir, requirementsURL string, log LogFunc) error {
	pythonExe := filepath.Join(installDir, "ovms", "python", "python.exe")
	if _, err := os.Stat(pythonExe); err != nil {
		return fmt.Errorf("OVMS Python not found at %s — prepare OVMS first", pythonExe)
	}

	log("Installing export requirements...")
	if err := runCmd(installDir, nil, log, pythonExe, "-m", "pip", "install", "--no-warn-script-location", "-r", requirementsURL); err != nil {
		return fmt.Errorf("install requirements: %w", err)
	}

	// Write marker so CheckStatus knows deps are ready.
	marker := filepath.Join(installDir, ".deps-ready")
	if err := os.WriteFile(marker, []byte("ready"), 0644); err != nil {
		return fmt.Errorf("write marker: %w", err)
	}

	log("Export environment ready.")
	return nil
}

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
