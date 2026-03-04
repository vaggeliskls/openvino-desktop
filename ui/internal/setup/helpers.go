package setup

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LogFunc is a callback for streaming output lines.
type LogFunc func(line string)

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractFileFromZip(zipPath, fileName, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == fileName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("%s not found in %s", fileName, zipPath)
}

func extractZip(zipPath, destDir string, log LogFunc) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		log("  " + f.Name)
	}
	return nil
}

// RunScript runs an arbitrary command in workDir, streaming output via log.
func RunScript(workDir string, log LogFunc, name string, args ...string) error {
	return runCmd(workDir, nil, log, name, args...)
}

// RunScriptEnv runs a command with a custom environment, streaming output via log.
func RunScriptEnv(workDir string, env []string, log LogFunc, name string, args ...string) error {
	return runCmd(workDir, env, log, name, args...)
}

func streamLines(r io.Reader, log LogFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, 4096)
	var partial string
	for {
		n, err := r.Read(buf)
		if n > 0 {
			partial += string(buf[:n])
			lines := strings.Split(partial, "\n")
			for _, line := range lines[:len(lines)-1] {
				line = strings.TrimRight(line, "\r")
				if line != "" {
					log(line)
				}
			}
			partial = lines[len(lines)-1]
		}
		if err != nil {
			if partial != "" {
				log(strings.TrimRight(partial, "\r"))
			}
			break
		}
	}
}
