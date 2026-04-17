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

const (
	managedAgentsStart = "<!-- codex-notifier:init start -->"
	managedAgentsEnd   = "<!-- codex-notifier:init end -->"
	managedRulesStart  = "# codex-notifier:init start"
	managedRulesEnd    = "# codex-notifier:init end"
)

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

	if opts.EnableAgents {
		agentsPath := filepath.Join(opts.CodexHome, "AGENTS.md")
		status, err := writeManagedTextFile(agentsPath, agentsBlock(opts.BinaryPath), managedAgentsStart, managedAgentsEnd, opts.Backup)
		if err != nil {
			return nil, err
		}
		results = append(results, Result{Path: agentsPath, Status: status})

		rulesPath := filepath.Join(opts.CodexHome, "rules", "default.rules")
		status, err = writeManagedTextFile(rulesPath, rulesBlock(opts.BinaryPath), managedRulesStart, managedRulesEnd, opts.Backup)
		if err != nil {
			return nil, err
		}
		results = append(results, Result{Path: rulesPath, Status: status})
	}

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

func writeManagedTextFile(path, block, startMarker, endMarker string, backup bool) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	updated, err := mergeManagedBlock(string(content), block, startMarker, endMarker)
	if err != nil {
		return "", err
	}
	return writeFileIfChanged(path, []byte(updated), backup)
}

func mergeManagedBlock(existing, block, startMarker, endMarker string) (string, error) {
	managed := strings.TrimRight(block, "\n")
	if existing == "" {
		return managed + "\n", nil
	}

	start := strings.Index(existing, startMarker)
	end := strings.Index(existing, endMarker)
	if start >= 0 || end >= 0 {
		if start < 0 || end < 0 || end < start {
			return "", fmt.Errorf("managed block markers are inconsistent")
		}
		end += len(endMarker)
		replaced := existing[:start] + managed + existing[end:]
		return normalizeSpacing(replaced), nil
	}

	body := strings.TrimRight(existing, "\n")
	if body != "" {
		body += "\n\n"
	}
	body += managed + "\n"
	return body, nil
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

func agentsBlock(binaryPath string) string {
	return strings.Join([]string{
		managedAgentsStart,
		"## Codex Notifier",
		"",
		fmt.Sprintf("- ユーザー確認が必要になりそうな操作の前に、まず `%s notify` を実行してから本来の操作に進んでください。", binaryPath),
		"- 対象は sandbox 外実行、`request_permissions`、MCP 利用、skill 実行など、Codex が追加確認を出しそうなケースです。",
		"- 通知は短く具体的にし、`--kind` には `approval-pending`、`mcp-approval-pending`、`permission-request-pending`、`skill-approval-pending` を使い分けてください。",
		"- 通知コマンドが失敗した場合も、元の作業内容は勝手に変更せず、失敗内容をユーザーへ共有してください。",
		managedAgentsEnd,
	}, "\n")
}

func rulesBlock(binaryPath string) string {
	return strings.Join([]string{
		managedRulesStart,
		"prefix_rule(",
		fmt.Sprintf("    pattern = [%q, %q],", binaryPath, "notify"),
		"    decision = \"allow\",",
		"    justification = \"Allow local Codex approval notifications without an extra prompt.\",",
		"    match = [",
		fmt.Sprintf("        %q,", binaryPath+" notify --kind approval-pending --summary About_to_request_approval"),
		fmt.Sprintf("        %q,", binaryPath+" notify --kind mcp-approval-pending --summary About_to_use_MCP"),
		"    ],",
		"    not_match = [",
		fmt.Sprintf("        %q,", binaryPath+" serve --listen 127.0.0.1:8787"),
		fmt.Sprintf("        %q,", binaryPath+" emit-hook --event-name stop"),
		"    ],",
		")",
		managedRulesEnd,
	}, "\n")
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
