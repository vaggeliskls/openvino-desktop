package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vaggeliskls/openvino-desk/ui/internal/setup"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets
var setupAssets embed.FS

const (
	defaultUvURL   = "https://github.com/astral-sh/uv/releases/download/0.10.8/uv-x86_64-pc-windows-msvc.zip"
	defaultOvmsURL = "https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_windows_python_on.zip"
)

// Config holds user-configurable settings.
var defaultSearchTags = []string{"OpenVINO", "Qwen", "Embedding"}
var defaultPipelineFilters = []string{"text-generation", "feature-extraction"}

type Config struct {
	InstallDir       string                 `json:"install_dir"`
	UvURL            string                 `json:"uv_url"`
	OvmsURL          string                 `json:"ovms_url"`
	StartupSet       bool                   `json:"startup_set"` // true once the startup preference has been written
	SearchTags       []string               `json:"search_tags"`
	PipelineFilters  []string               `json:"pipeline_filters"`
	SearchLimit      int                    `json:"search_limit"`
	TextGenExport    map[string]interface{} `json:"text_gen_export"`   // Flexible export options for text-generation
	EmbeddingsExport map[string]interface{} `json:"embeddings_export"` // Flexible export options for embeddings
}

// StatusResult reports whether each component is ready.
type StatusResult struct {
	UvReady     bool   `json:"uv_ready"`
	DepsReady   bool   `json:"deps_ready"`
	OvmsReady   bool   `json:"ovms_ready"`
	OvmsVersion string `json:"ovms_version"`
}

// ovmsVersionFromURL extracts the version tag from an OVMS release URL.
// e.g. ".../download/v2026.0/ovms_windows..." → "2026.0"
func ovmsVersionFromURL(ovmsURL string) string {
	for _, part := range strings.Split(ovmsURL, "/") {
		if strings.HasPrefix(part, "v") && len(part) > 1 {
			return part[1:]
		}
	}
	return ""
}

// App is the Wails application struct.
type App struct {
	ctx      context.Context
	config   Config
	ovmsProc *exec.Cmd
	ovmsMu   sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
	// On first run, register the app to start with Windows by default.
	if !a.config.StartupSet {
		a.SetStartup(true) //nolint: errcheck — best-effort on first run
		a.config.StartupSet = true
		a.SaveConfig(a.config) //nolint: errcheck
	}
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openvino-desk", "config.json")
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		InstallDir:      filepath.Join(home, "openvino-desk"),
		UvURL:           defaultUvURL,
		OvmsURL:         defaultOvmsURL,
		SearchTags:      defaultSearchTags,
		PipelineFilters: defaultPipelineFilters,
		SearchLimit:     30,
		TextGenExport: map[string]interface{}{
			"target_device":          "GPU",
			"cache":                  2,
			"kv_cache_precision":     "u8",
			"enable_prefix_caching":  true,
			"max_num_batched_tokens": 2048,
			"max_num_seqs":           8,
		},
		EmbeddingsExport: map[string]interface{}{
			"target_device":             "CPU",
			"weight_format":             "fp16",
			"extra_quantization_params": "--library sentence_transformers",
		},
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
		return
	}
	// Fill in URL defaults for older configs that predate these fields.
	if a.config.UvURL == "" {
		a.config.UvURL = defaultUvURL
	}
	if a.config.OvmsURL == "" {
		a.config.OvmsURL = defaultOvmsURL
	}
	if len(a.config.SearchTags) == 0 {
		a.config.SearchTags = defaultSearchTags
	}
	if len(a.config.PipelineFilters) == 0 {
		a.config.PipelineFilters = defaultPipelineFilters
	}
	if a.config.SearchLimit == 0 {
		a.config.SearchLimit = 30
	}
	// Apply defaults for text_gen_export if empty
	if len(a.config.TextGenExport) == 0 {
		a.config.TextGenExport = map[string]interface{}{
			"target_device":          "GPU",
			"cache":                  2,
			"kv_cache_precision":     "u8",
			"enable_prefix_caching":  true,
			"max_num_batched_tokens": 2048,
			"max_num_seqs":           8,
		}
	}
	// Apply defaults for embeddings_export if empty
	if len(a.config.EmbeddingsExport) == 0 {
		a.config.EmbeddingsExport = map[string]interface{}{
			"target_device":             "CPU",
			"weight_format":             "fp16",
			"extra_quantization_params": "--library sentence_transformers",
		}
	}
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() Config {
	return a.config
}

// SaveConfig persists the configuration to disk.
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

// CheckStatus reports whether uv, the export venv, and OVMS are present.
func (a *App) CheckStatus() StatusResult {
	uvBin := filepath.Join(a.config.InstallDir, "uv.exe")
	venvPython := filepath.Join(a.config.InstallDir, "export", "Scripts", "python.exe")
	ovmsDir := filepath.Join(a.config.InstallDir, "ovms")

	_, uvErr := os.Stat(uvBin)
	_, depsErr := os.Stat(venvPython)
	_, ovmsErr := os.Stat(ovmsDir)

	return StatusResult{
		UvReady:     uvErr == nil,
		DepsReady:   depsErr == nil,
		OvmsReady:   ovmsErr == nil,
		OvmsVersion: ovmsVersionFromURL(a.config.OvmsURL),
	}
}

// extractAssets writes embedded assets (requirements, scripts) to installDir,
// skipping uv.exe which is downloaded from the configured URL instead.
func (a *App) extractAssets() error {
	return fs.WalkDir(setupAssets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Base(path) == "uv.exe" {
			return nil // uv is downloaded from URL, not embedded
		}
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

// PrepareExport extracts bundled assets, downloads uv, then sets up the Python environment.
func (a *App) PrepareExport() error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	if a.config.UvURL == "" {
		return fmt.Errorf("uv download URL is not configured")
	}
	a.emit("Extracting bundled assets...")
	if err := a.extractAssets(); err != nil {
		return fmt.Errorf("extract assets: %w", err)
	}
	return setup.PrepareExport(a.config.InstallDir, a.config.UvURL, a.emit)
}

// HFModel is a minimal representation of a Hugging Face model search result.
type HFModel struct {
	ID          string `json:"id"`
	PipelineTag string `json:"pipeline_tag"`
	Downloads   int    `json:"downloads"`
	Likes       int    `json:"likes"`
	LibraryName string `json:"library_name"`
}

func hfGet(endpoint string) ([]HFModel, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("huggingface request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("huggingface API: HTTP %d", resp.StatusCode)
	}
	var models []HFModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return models, nil
}

// SearchModels queries the Hugging Face API with the given pipeline filters.
// Pass an empty slice to search without any pipeline restriction.
// One request is made per filter and results are merged/deduplicated.
func (a *App) SearchModels(query string, filters []string) ([]HFModel, error) {
	base := fmt.Sprintf("https://huggingface.co/api/models?limit=%d&sort=downloads&direction=-1", a.config.SearchLimit)
	if query != "" {
		base += "&search=" + url.QueryEscape(query)
	}
	if len(filters) == 0 {
		return hfGet(base)
	}
	seen := map[string]bool{}
	var merged []HFModel
	for _, f := range filters {
		results, err := hfGet(base + "&pipeline_tag=" + url.QueryEscape(f))
		if err != nil {
			return nil, err
		}
		for _, m := range results {
			if !seen[m.ID] {
				seen[m.ID] = true
				merged = append(merged, m)
			}
		}
	}
	return merged, nil
}

// PullModel downloads an OpenVINO model from Hugging Face using huggingface_hub.
func (a *App) PullModel(modelID string) error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	if modelID == "" {
		return fmt.Errorf("no model selected")
	}

	ovmsDir := filepath.Join(a.config.InstallDir, "ovms")
	ovmsExe := filepath.Join(ovmsDir, "ovms.exe")

	if _, err := os.Stat(ovmsExe); err != nil {
		return fmt.Errorf("ovms.exe not found at %s", ovmsExe)
	}

	modelsDir := filepath.Join(a.config.InstallDir, "models")
	os.MkdirAll(modelsDir, 0755) //nolint: errcheck

	// Use OVMS CLI to pull the model
	// ovms --pull --source_model <model> --model_repository_path <path> --model_name <name>
	cmd := exec.Command(ovmsExe,
		"--pull",
		"--source_model", modelID,
		"--model_repository_path", modelsDir,
		"--model_name", modelID,
	)
	cmd.Dir = ovmsDir
	cmd.Env = buildOVMSEnv(ovmsDir)

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
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				runtime.EventsEmit(a.ctx, "log", line)
			}
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				runtime.EventsEmit(a.ctx, "log", line)
			}
		}
	}()
	wg.Wait()

	return cmd.Wait()
}

// venvEnv returns os.Environ() with the venv Scripts directory prepended to PATH,
// so that subprocesses spawned by export_model.py (e.g. optimum-cli) can be found.
func (a *App) venvEnv() []string {
	scriptsDir := filepath.Join(a.config.InstallDir, "export", "Scripts")
	base := os.Environ()
	result := make([]string, 0, len(base))
	for _, e := range base {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			result = append(result, "PATH="+scriptsDir+";"+e[5:])
		} else {
			result = append(result, e)
		}
	}
	return result
}

// ExportTextGen exports a text-generation model using export_model.py text_generation.
func (a *App) ExportTextGen(modelID string, opts map[string]interface{}) error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	if modelID == "" {
		return fmt.Errorf("no model selected")
	}
	venvPython := filepath.Join(a.config.InstallDir, "export", "Scripts", "python.exe")
	scriptPath := filepath.Join(a.config.InstallDir, "export-model-requirements", "export_model.py")
	modelsDir := filepath.Join(a.config.InstallDir, "models")
	args := []string{
		scriptPath, "text_generation",
		"--source_model", modelID,
		"--model_name", modelID,
		"--model_repository_path", modelsDir,
	}

	// Add options dynamically from the map
	args = a.buildExportArgs(args, opts)

	if err := setup.RunScriptEnv(a.config.InstallDir, a.venvEnv(), a.emit, venvPython, args...); err != nil {
		return err
	}
	// Update config.json with target_device after export completes
	if targetDevice, ok := opts["target_device"].(string); ok {
		return a.updateModelTargetDevice(modelID, targetDevice)
	}
	return nil
}

// ExportEmbeddings exports an embeddings model using export_model.py embeddings_ov.
func (a *App) ExportEmbeddings(modelID string, opts map[string]interface{}) error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	if modelID == "" {
		return fmt.Errorf("no model selected")
	}
	venvPython := filepath.Join(a.config.InstallDir, "export", "Scripts", "python.exe")
	scriptPath := filepath.Join(a.config.InstallDir, "export-model-requirements", "export_model.py")
	modelsDir := filepath.Join(a.config.InstallDir, "models")
	args := []string{
		scriptPath, "embeddings_ov",
		"--source_model", modelID,
		"--model_name", modelID,
		"--model_repository_path", modelsDir,
	}

	// Add options dynamically from the map
	args = a.buildExportArgs(args, opts)

	if err := setup.RunScriptEnv(a.config.InstallDir, a.venvEnv(), a.emit, venvPython, args...); err != nil {
		return err
	}
	// Update config.json with target_device after export completes
	if targetDevice, ok := opts["target_device"].(string); ok {
		return a.updateModelTargetDevice(modelID, targetDevice)
	}
	return nil
}

// buildExportArgs converts a map of options to command-line arguments.
func (a *App) buildExportArgs(baseArgs []string, opts map[string]interface{}) []string {
	args := baseArgs
	for key, value := range opts {
		switch v := value.(type) {
		case bool:
			if v {
				args = append(args, "--"+key)
			}
		case int:
			args = append(args, "--"+key, strconv.Itoa(v))
		case float64:
			args = append(args, "--"+key, strconv.Itoa(int(v)))
		case string:
			if v != "" {
				args = append(args, "--"+key, v)
			}
		default:
			// Skip unknown types
		}
	}
	return args
}

// updateModelTargetDevice updates the target_device field for a model in config.json.
func (a *App) updateModelTargetDevice(modelName, targetDevice string) error {
	configPath := filepath.Join(a.config.InstallDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If config doesn't exist yet, that's ok - it will be created by OVMS
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config.json: %w", err)
	}

	var ovmsConfig OVMSConfig
	if err := json.Unmarshal(data, &ovmsConfig); err != nil {
		return fmt.Errorf("parse config.json: %w", err)
	}

	// Find and update the model's target_device
	updated := false
	for i := range ovmsConfig.ModelConfigList {
		if ovmsConfig.ModelConfigList[i].Config.Name == modelName {
			ovmsConfig.ModelConfigList[i].Config.TargetDevice = targetDevice
			updated = true
			break
		}
	}

	if !updated {
		// Model not found in config yet - this is normal, will be added later
		return nil
	}

	// Write updated config
	updatedData, err := json.MarshalIndent(ovmsConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config.json: %w", err)
	}
	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	return nil
}

// ResetExport removes the uv binary, Python installation and export venv.
func (a *App) ResetExport() error {
	dirs := []string{
		filepath.Join(a.config.InstallDir, "uv.exe"),
		filepath.Join(a.config.InstallDir, "python"),
		filepath.Join(a.config.InstallDir, "export"),
	}
	for _, d := range dirs {
		if err := os.RemoveAll(d); err != nil {
			return fmt.Errorf("remove %s: %w", d, err)
		}
	}
	return nil
}

// ResetOVMS removes the OVMS server directory.
func (a *App) ResetOVMS() error {
	ovmsDir := filepath.Join(a.config.InstallDir, "ovms")
	if err := os.RemoveAll(ovmsDir); err != nil {
		return fmt.Errorf("remove ovms: %w", err)
	}
	return nil
}

// PrepareOVMS downloads and extracts the OVMS server.
func (a *App) PrepareOVMS() error {
	if a.config.InstallDir == "" {
		return fmt.Errorf("install directory is not configured")
	}
	if a.config.OvmsURL == "" {
		return fmt.Errorf("OVMS download URL is not configured")
	}
	return setup.PrepareOVMS(a.config.InstallDir, a.config.OvmsURL, a.emit)
}

// buildOVMSEnv constructs the process environment for running ovms.exe,
// replicating what setupvars.ps1 does (sets OVMS_DIR, PYTHONHOME, prepends PATH).
func buildOVMSEnv(ovmsDir string) []string {
	var prepend []string
	pythonDir := filepath.Join(ovmsDir, "python")
	if _, err := os.Stat(pythonDir); err == nil {
		prepend = []string{ovmsDir, pythonDir, filepath.Join(pythonDir, "Scripts")}
	} else {
		prepend = []string{ovmsDir}
	}

	base := os.Environ()
	result := make([]string, 0, len(base)+2)
	for _, e := range base {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			result = append(result, "PATH="+strings.Join(prepend, ";")+";"+e[5:])
		} else {
			result = append(result, e)
		}
	}
	result = append(result, "OVMS_DIR="+ovmsDir)
	if _, err := os.Stat(pythonDir); err == nil {
		result = append(result, "PYTHONHOME="+pythonDir)
	}
	return result
}

func (a *App) emitServerLog(line string) {
	runtime.EventsEmit(a.ctx, "server-log", line)
}

func (a *App) streamReader(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			a.emitServerLog(line)
		}
	}
}

// StartOVMS sources setupvars.ps1 env and starts the OVMS server process,
// streaming its output via "server-log" events.
func (a *App) StartOVMS() error {
	a.ovmsMu.Lock()
	defer a.ovmsMu.Unlock()

	if a.ovmsProc != nil {
		return fmt.Errorf("OVMS server is already running")
	}

	ovmsDir := filepath.Join(a.config.InstallDir, "ovms")
	ovmsExe := filepath.Join(ovmsDir, "ovms.exe")
	ovmsConfig := filepath.Join(a.config.InstallDir, "config.json")
	if _, err := os.Stat(ovmsExe); err != nil {
		return fmt.Errorf("ovms.exe not found at %s", ovmsExe)
	}

	modelsDir := filepath.Join(a.config.InstallDir, "models")
	os.MkdirAll(modelsDir, 0755) //nolint: errcheck

	cmd := exec.Command(ovmsExe, "--port", "9000", "--rest_port", "8080", "--config_path", ovmsConfig)
	cmd.Dir = ovmsDir
	cmd.Env = buildOVMSEnv(ovmsDir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ovms: %w", err)
	}

	a.ovmsProc = cmd
	runtime.EventsEmit(a.ctx, "server-status", true)

	go func() {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); a.streamReader(stdout) }()
		go func() { defer wg.Done(); a.streamReader(stderr) }()
		wg.Wait()
		cmd.Wait() //nolint: errcheck

		a.ovmsMu.Lock()
		a.ovmsProc = nil
		a.ovmsMu.Unlock()
		runtime.EventsEmit(a.ctx, "server-status", false)
	}()

	return nil
}

// StopOVMS kills the running OVMS server process.
func (a *App) StopOVMS() error {
	a.ovmsMu.Lock()
	defer a.ovmsMu.Unlock()

	if a.ovmsProc == nil {
		return nil
	}
	if err := a.ovmsProc.Process.Kill(); err != nil {
		return fmt.Errorf("kill ovms: %w", err)
	}
	return nil
}

// IsOVMSRunning reports whether the OVMS server process is active.
func (a *App) IsOVMSRunning() bool {
	a.ovmsMu.Lock()
	defer a.ovmsMu.Unlock()
	return a.ovmsProc != nil
}

// ModelInfo represents an installed model with its configuration.
type ModelInfo struct {
	Name         string `json:"name"`
	BasePath     string `json:"base_path"`
	TargetDevice string `json:"target_device"`
}

// OVMSConfig matches the structure of config.json used by OVMS.
type OVMSConfig struct {
	ModelConfigList []struct {
		Config struct {
			Name         string                 `json:"name"`
			BasePath     string                 `json:"base_path"`
			TargetDevice string                 `json:"target_device"`
			PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`
			Nireq        int                    `json:"nireq,omitempty"`
		} `json:"config"`
	} `json:"model_config_list"`
}

// GetInstalledModels returns the list of models from config.json.
func (a *App) GetInstalledModels() ([]ModelInfo, error) {
	configPath := filepath.Join(a.config.InstallDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ModelInfo{}, nil
		}
		return nil, fmt.Errorf("read config.json: %w", err)
	}

	var ovmsConfig OVMSConfig
	if err := json.Unmarshal(data, &ovmsConfig); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}

	models := make([]ModelInfo, 0, len(ovmsConfig.ModelConfigList))
	for _, item := range ovmsConfig.ModelConfigList {
		models = append(models, ModelInfo{
			Name:         item.Config.Name,
			BasePath:     item.Config.BasePath,
			TargetDevice: item.Config.TargetDevice,
		})
	}
	return models, nil
}

// DeleteInstalledModel removes a model from config.json and deletes its files.
func (a *App) DeleteInstalledModel(modelName string) error {
	if modelName == "" {
		return fmt.Errorf("model name is required")
	}

	configPath := filepath.Join(a.config.InstallDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config.json: %w", err)
	}

	var ovmsConfig OVMSConfig
	if err := json.Unmarshal(data, &ovmsConfig); err != nil {
		return fmt.Errorf("parse config.json: %w", err)
	}

	// Find and remove the model from config
	var modelPath string
	newList := make([]struct {
		Config struct {
			Name         string                 `json:"name"`
			BasePath     string                 `json:"base_path"`
			TargetDevice string                 `json:"target_device"`
			PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`
			Nireq        int                    `json:"nireq,omitempty"`
		} `json:"config"`
	}, 0)

	for _, item := range ovmsConfig.ModelConfigList {
		if item.Config.Name == modelName {
			modelPath = item.Config.BasePath
		} else {
			newList = append(newList, item)
		}
	}

	if modelPath == "" {
		return fmt.Errorf("model %q not found in config.json", modelName)
	}

	// Delete model directory first (convert Windows path separator if needed)
	modelPath = filepath.FromSlash(strings.ReplaceAll(modelPath, "\\", "/"))
	if !filepath.IsAbs(modelPath) {
		modelPath = filepath.Join(a.config.InstallDir, modelPath)
	}
	if err := os.RemoveAll(modelPath); err != nil {
		return fmt.Errorf("remove model directory: %w", err)
	}

	// Update config.json after successful deletion
	ovmsConfig.ModelConfigList = newList
	updatedData, err := json.MarshalIndent(ovmsConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config.json: %w", err)
	}
	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	return nil
}
