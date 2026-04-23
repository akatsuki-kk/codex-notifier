package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const ()

type Options struct {
	CodexHome      string
	BinaryPath     string
	EnableAgents   bool
	EnableStopHook bool
	Backup         bool
	ServerURL      string
}

type Result struct {
	Path   string
	Status string
}

func Run(opts Options) ([]Result, error) {
	if strings.TrimSpace(opts.CodexHome) == "" {
		return nil, fmt.Errorf("codex home is required")
	}
	if strings.TrimSpace(opts.BinaryPath) == "" {
		return nil, fmt.Errorf("binary path is required")
	}
	if strings.TrimSpace(opts.ServerURL) == "" {
		return nil, fmt.Errorf("server url is required")
	}

	var results []Result

	configResults, err := updateConfig(opts)
	if err != nil {
		return nil, err
	}
	results = append(results, configResults...)

	if opts.EnableStopHook {
		hooksPath := filepath.Join(opts.CodexHome, "hooks.json")
		status, err := writeHooksFile(hooksPath, opts, opts.Backup)
		if err != nil {
			return nil, err
		}
		results = append(results, Result{Path: hooksPath, Status: status})
	}

	return results, nil
}

func updateConfig(opts Options) ([]Result, error) {
	configPath := filepath.Join(opts.CodexHome, "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config.toml: %w", err)
	}

	updated := mergeConfigToml(string(content), opts.EnableStopHook)
	status, err := writeFileIfChanged(configPath, []byte(updated), opts.Backup)
	if err != nil {
		return nil, err
	}
	return []Result{{Path: configPath, Status: status}}, nil
}

func mergeConfigToml(content string, enableStopHook bool) string {
	if !enableStopHook {
		return content
	}

	lines := splitLines(content)
	featuresStart := -1
	featuresEnd := len(lines)
	codexHooksLine := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSectionHeader(trimmed) {
			if trimmed == "[features]" {
				featuresStart = i
				continue
			}
			if featuresStart >= 0 {
				featuresEnd = i
				break
			}
		}
		if featuresStart >= 0 && strings.HasPrefix(trimmed, "codex_hooks") {
			codexHooksLine = i
		}
	}

	if featuresStart == -1 {
		body := strings.TrimRight(content, "\n")
		if body != "" {
			body += "\n\n"
		}
		body += "[features]\n"
		body += "codex_hooks = true\n"
		return body
	}

	if codexHooksLine >= 0 {
		lines[codexHooksLine] = "codex_hooks = true"
		return normalizeSpacing(strings.Join(lines, "\n"))
	}

	insertAt := featuresEnd
	lines = slices.Insert(lines, insertAt, "codex_hooks = true")
	return normalizeSpacing(strings.Join(lines, "\n"))
}

func writeHooksFile(path string, opts Options, backup bool) (string, error) {
	type hooksFile struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}

	var current hooksFile
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read hooks.json: %w", err)
	}
	if len(bytes.TrimSpace(content)) > 0 {
		if err := json.Unmarshal(content, &current); err != nil {
			return "", fmt.Errorf("parse hooks.json: %w", err)
		}
	}
	if current.Hooks == nil {
		current.Hooks = map[string]json.RawMessage{}
	}

	stopPayload := []map[string]any{
		{
			"hooks": []map[string]any{
				{
					"type":    "command",
					"command": fmt.Sprintf("%s emit-hook --server %s --event-name stop", opts.BinaryPath, opts.ServerURL),
					"timeout": 30,
				},
			},
		},
	}
	raw, err := json.Marshal(stopPayload)
	if err != nil {
		return "", fmt.Errorf("marshal stop hook: %w", err)
	}
	current.Hooks["Stop"] = raw

	formatted, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal hooks.json: %w", err)
	}
	formatted = append(formatted, '\n')
	return writeFileIfChanged(path, formatted, backup)
}

func writeFileIfChanged(path string, content []byte, backup bool) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if bytes.Equal(existing, content) {
		return "unchanged", nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if backup && len(existing) > 0 {
		backupPath := path + ".bak"
		if err := os.WriteFile(backupPath, existing, 0o644); err != nil {
			return "", fmt.Errorf("write backup %s: %w", backupPath, err)
		}
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	if len(existing) == 0 {
		return "created", nil
	}
	return "updated", nil
}

func isSectionHeader(value string) bool {
	matched, _ := regexp.MatchString(`^\[[^]]+\]$`, value)
	return matched
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func normalizeSpacing(value string) string {
	lines := splitLines(value)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
