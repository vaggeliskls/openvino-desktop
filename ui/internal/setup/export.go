package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const pythonVer = "3.12.12"

// PrepareExport installs Python, creates a venv and installs requirements.
// installDir is the base directory (e.g. C:\Users\user\openvino-desk).
// uv.exe and requirements.txt are expected to already be present in installDir
// (extracted from embedded assets by the caller).
func PrepareExport(installDir string, log LogFunc) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	uvBin := filepath.Join(installDir, "uv.exe")
	requirementsFile := filepath.Join(installDir, "export-model-requirements", "requirements.txt")

	// Step 1: Install Python
	pythonExe := filepath.Join(installDir, "python", "cpython-"+pythonVer+"-windows-x86_64-none", "python.exe")
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		log("Installing Python " + pythonVer + "...")
		pythonInstallDir := filepath.Join(installDir, "python")
		if err := runCmd(installDir, log, uvBin, "python", "install", pythonVer, "--install-dir", pythonInstallDir); err != nil {
			return fmt.Errorf("install python: %w", err)
		}
	} else {
		log("Python already installed, skipping.")
	}

	// Step 2: Create venv
	venvDir := filepath.Join(installDir, "export")
	venvPython := filepath.Join(venvDir, "Scripts", "python.exe")
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		log("Creating virtual environment...")
		if err := runCmd(installDir, log, uvBin, "venv", venvDir, "--python", pythonExe, "--relocatable"); err != nil {
			return fmt.Errorf("create venv: %w", err)
		}
	} else {
		log("Venv already exists, skipping.")
	}

	// Step 3: Install requirements
	log("Installing requirements...")
	if err := runCmd(installDir, log, uvBin, "pip", "install", "--python", venvPython, "-r", requirementsFile); err != nil {
		return fmt.Errorf("install requirements: %w", err)
	}

	log("Export environment ready.")
	return nil
}

func runCmd(workDir string, log LogFunc, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workDir

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
