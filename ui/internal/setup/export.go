package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const pythonVer = "3.12.12"

// PrepareExport downloads uv if needed, installs Python, creates a venv and installs requirements.
// installDir is the base directory (e.g. C:\Users\user\openvino-desk).
// uvURL is the download URL for the uv zip archive (contains uv.exe).
func PrepareExport(installDir, uvURL string, log LogFunc) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	uvBin := filepath.Join(installDir, "uv.exe")
	requirementsFile := filepath.Join(installDir, "export-model-requirements", "requirements.txt")

	// Step 0: Download uv if not present
	if _, err := os.Stat(uvBin); os.IsNotExist(err) {
		log("Downloading uv...")
		tmpZip := filepath.Join(installDir, "uv-tmp.zip")
		if err := downloadFile(uvURL, tmpZip); err != nil {
			return fmt.Errorf("download uv: %w", err)
		}
		if err := extractFileFromZip(tmpZip, "uv.exe", uvBin); err != nil {
			os.Remove(tmpZip)
			return fmt.Errorf("extract uv.exe: %w", err)
		}
		os.Remove(tmpZip)
		log("uv ready.")
	} else {
		log("uv already present, skipping download.")
	}

	// Step 1: Install Python
	pythonExe := filepath.Join(installDir, "python", "cpython-"+pythonVer+"-windows-x86_64-none", "python.exe")
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		log("Installing Python " + pythonVer + "...")
		pythonInstallDir := filepath.Join(installDir, "python")
		if err := runCmd(installDir, nil, log, uvBin, "python", "install", pythonVer, "--install-dir", pythonInstallDir); err != nil {
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
		if err := runCmd(installDir, nil, log, uvBin, "venv", venvDir, "--python", pythonExe, "--relocatable"); err != nil {
			return fmt.Errorf("create venv: %w", err)
		}
	} else {
		log("Venv already exists, skipping.")
	}

	// Step 3: Install requirements
	log("Installing requirements...")
	if err := runCmd(installDir, nil, log, uvBin, "pip", "install", "--python", venvPython, "-r", requirementsFile); err != nil {
		return fmt.Errorf("install requirements: %w", err)
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
