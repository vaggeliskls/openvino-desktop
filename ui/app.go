package main

import (
	"context"
	"encoding/json"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vaggeliskls/openvino-desk/ui/internal/setup"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets
var setupAssets embed.FS

// Config holds user-configurable settings.
type Config struct {
	InstallDir string `json:"install_dir"`
}

// App is the Wails application struct.
type App struct {
	ctx    context.Context
	config Config
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openvino-desk", "config.json")
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		InstallDir: filepath.Join(home, "openvino-desk"),
	}
}

func (a *App) loadConfig() {
	data, err := os.ReadFile(configPath())
	if err != nil {
		a.config = defaultConfig()
		return
	}
	if err := json.Unmarshal(data, &a.config); err != nil {
		a.config = defaultConfig()
	}
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() Config {
	return a.config
}

// SaveConfig saves the configuration to disk.
func (a *App) SaveConfig(config Config) error {
	a.config = config
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// extractAssets writes embedded assets (uv.exe, requirements, scripts) to installDir.
func (a *App) extractAssets() error {
	return fs.WalkDir(setupAssets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Compute destination path relative to installDir
		rel, _ := filepath.Rel("assets", path)
		dest := filepath.Join(a.config.InstallDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		data, err := setupAssets.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0755)
	})
}

func (a *App) emit(line string) {
	runtime.EventsEmit(a.ctx, "log", line)
}

// PrepareExport extracts embedded assets then runs the export environment setup.
func (a *App) PrepareExport() error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	a.emit("Extracting bundled assets...")
	if err := a.extractAssets(); err != nil {
		return fmt.Errorf("extract assets: %w", err)
	}
	return setup.PrepareExport(a.config.InstallDir, a.emit)
}

// PrepareOVMS runs the OVMS server download and extraction.
func (a *App) PrepareOVMS() error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	return setup.PrepareOVMS(a.config.InstallDir, a.emit)
}
