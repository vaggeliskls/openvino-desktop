package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ovmsURL    = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_windows_python_on.zip"
	ovmsTmpZip = "ovms-tmp.zip"
	ovmsDir    = "ovms"
)

// PrepareOVMS downloads and extracts the OVMS server into installDir.
// The zip already contains an ovms/ folder internally.
func PrepareOVMS(installDir string, log LogFunc) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	ovmsDest := filepath.Join(installDir, ovmsDir)
	if _, err := os.Stat(ovmsDest); err == nil {
		log("Removing existing ovms/...")
		if err := os.RemoveAll(ovmsDest); err != nil {
			return fmt.Errorf("remove ovms: %w", err)
		}
	}

	tmpZip := filepath.Join(installDir, ovmsTmpZip)
	log("Downloading OVMS...")
	if err := downloadFile(ovmsURL, tmpZip); err != nil {
		return fmt.Errorf("download ovms: %w", err)
	}

	log("Extracting OVMS...")
	if err := extractZip(tmpZip, installDir, log); err != nil {
		os.Remove(tmpZip)
		return fmt.Errorf("extract ovms: %w", err)
	}

	os.Remove(tmpZip)
	log("OVMS ready at " + ovmsDest)
	return nil
}
