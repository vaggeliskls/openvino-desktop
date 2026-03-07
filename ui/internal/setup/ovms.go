package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ovmsTmpZip = "ovms-tmp.zip"
	ovmsDir    = "ovms"
)

// PrepareOVMS downloads and extracts the OVMS server into installDir.
func PrepareOVMS(installDir, ovmsURL string, log LogFunc) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	ovmsDest := filepath.Join(installDir, ovmsDir)
	if _, err := os.Stat(ovmsDest); err == nil {
		log("Removing existing ovms/...")
		if err := removeDir(ovmsDest); err != nil {
			return fmt.Errorf("remove ovms: %w", err)
		}
	}

	tmpZip := filepath.Join(installDir, ovmsTmpZip)
	log("Downloading OVMS...")
	if err := downloadFile(ovmsURL, tmpZip); err != nil {
		return fmt.Errorf("download ovms: %w", err)
	}

	log("Extracting OVMS...")
	if err := unzip(tmpZip, installDir); err != nil {
		os.Remove(tmpZip)
		return fmt.Errorf("extract ovms: %w", err)
	}
	os.Remove(tmpZip)
	log("OVMS ready at " + ovmsDest)
	return nil
}

// PrepareExport creates a uv venv and installs the export requirements into it,
// then writes the .deps-ready marker.
// uvDownloadURL overrides the default uv download URL when non-empty.
func PrepareExport(installDir, uvDownloadURL string, log LogFunc) error {
	uvExe := UVExePath(installDir)
	ovmsPython := OVMSPythonPath(installDir)
	requirementsPath := filepath.Join(installDir, "export-model-requirements", "requirements.txt")

	downloadURL := DefaultUVURL()
	if uvDownloadURL != "" {
		downloadURL = uvDownloadURL
	}

	if _, err := os.Stat(uvExe); err != nil {
		log("Downloading uv...")
		if err := downloadUVBinary(downloadURL, uvExe); err != nil {
			return fmt.Errorf("download uv: %w", err)
		}
	}
	if _, err := os.Stat(ovmsPython); err != nil {
		return fmt.Errorf("python not found at %s — run Prepare OVMS first", ovmsPython)
	}
	if _, err := os.Stat(requirementsPath); err != nil {
		return fmt.Errorf("requirements.txt not found at %s", requirementsPath)
	}

	venvDir := filepath.Join(installDir, "export")
	log("Creating export venv...")
	if err := RunScript(installDir, log, uvExe, "venv", venvDir, "--python", ovmsPython); err != nil {
		return fmt.Errorf("uv venv: %w", err)
	}

	log("Installing export dependencies...")
	if err := RunScript(installDir, log, uvExe, "pip", "install", "--python", ovmsPython, "-r", requirementsPath); err != nil {
		return fmt.Errorf("uv pip install: %w", err)
	}

	marker := filepath.Join(installDir, ".deps-ready")
	if err := os.WriteFile(marker, []byte("ready"), 0644); err != nil {
		return fmt.Errorf("write deps marker: %w", err)
	}

	log("Export environment ready.")
	return nil
}
