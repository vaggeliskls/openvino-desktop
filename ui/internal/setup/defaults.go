package setup

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ovmsURLWindows  = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_windows_python_on.zip"
	ovmsURLRedHat   = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_redhat_python_on.tar.gz"
	ovmsURLUbuntu24 = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_ubuntu24_python_on.tar.gz"
	ovmsURLUbuntu22 = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_ubuntu22_python_on.tar.gz"

	uvURLWindows = "https://github.com/turintech/openvino-desktop/releases/download/uv/uv.exe"
	uvURLLinux   = "https://github.com/astral-sh/uv/releases/download/0.10.9/uv-x86_64-unknown-linux-gnu.tar.gz"
)

// ExeSuffix returns ".exe" on Windows, empty string on other OS.
func ExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// UVExePath returns the path to the uv executable inside installDir.
func UVExePath(installDir string) string {
	return filepath.Join(installDir, "uv"+ExeSuffix())
}

// OVMSPythonPath returns the Python interpreter for OVMS operations.
// On Windows: the bundled Python inside the OVMS directory.
// On Linux: the system Python3.
func OVMSPythonPath(installDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(installDir, ovmsDir, "python", "python.exe")
	}
	return "/usr/bin/python3"
}

// OVMSLibPath returns the path to the OVMS shared libraries directory.
func OVMSLibPath(installDir string) string {
	return filepath.Join(installDir, ovmsDir, "lib")
}

// DefaultUVURL returns the default uv download URL for the current OS.
func DefaultUVURL() string {
	switch runtime.GOOS {
	case "windows":
		return uvURLWindows
	default:
		return uvURLLinux
	}
}

// DefaultOVMSURL returns the default OVMS download URL for the current OS.
// On Linux it reads /etc/os-release to distinguish between distros.
func DefaultOVMSURL() string {
	switch runtime.GOOS {
	case "windows":
		return ovmsURLWindows
	case "linux":
		return linuxOVMSURL()
	default:
		return ovmsURLUbuntu22
	}
}

func linuxOVMSURL() string {
	info := parseOSRelease()

	id := strings.ToLower(info["ID"])
	idLike := strings.ToLower(info["ID_LIKE"])
	version := info["VERSION_ID"]

	// RedHat-based
	if id == "rhel" || id == "centos" || id == "fedora" ||
		strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
		return ovmsURLRedHat
	}

	// Ubuntu
	if id == "ubuntu" || strings.Contains(idLike, "ubuntu") {
		if strings.HasPrefix(version, "24") {
			return ovmsURLUbuntu24
		}
		return ovmsURLUbuntu22
	}

	// Debian-based fallback
	return ovmsURLUbuntu22
}

// parseOSRelease reads key=value pairs from /etc/os-release.
func parseOSRelease() map[string]string {
	result := make(map[string]string)
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return result
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		result[key] = val
	}
	return result
}
