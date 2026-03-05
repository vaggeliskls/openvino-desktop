package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ovmsTmpZip    = "ovms-tmp.zip"
	exportTmpZip  = "export-tmp.zip"
	ovmsDir       = "ovms"
	exportZipURL  = "https://github.com/vaggeliskls/openvino-desk/releases/download/export-v2026.0/export-v2026.0-windows-x64.zip"
)

// PrepareOVMS downloads and extracts the OVMS server into installDir,
// then downloads the export bundle (site-packages + export_model.py) and
// installs it into the OVMS Python environment.
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

	exportTmp := filepath.Join(installDir, exportTmpZip)
	log("Downloading export bundle...")
	if err := downloadFile(exportZipURL, exportTmp); err != nil {
		return fmt.Errorf("download export bundle: %w", err)
	}

	pythonLibDir := filepath.Join(ovmsDest, "python", "Lib")
	log("Installing export bundle...")
	if err := unzip(exportTmp, pythonLibDir); err != nil {
		os.Remove(exportTmp)
		return fmt.Errorf("extract export bundle: %w", err)
	}
	os.Remove(exportTmp)

	marker := filepath.Join(installDir, ".deps-ready")
	if err := os.WriteFile(marker, []byte("ready"), 0644); err != nil {
		return fmt.Errorf("write deps marker: %w", err)
	}

	log("Setup complete.")
	return nil
}
