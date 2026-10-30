package src

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const claudeModel = "haiku"

// claudeCandidates lists the usual installation locations. macOS GUI apps
// launch with a minimal PATH, so command lookup alone is not enough.
func claudeCandidates() []string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".claude", "local", "claude"),
		filepath.Join(home, ".npm-global", "bin", "claude"),
		filepath.Join(home, ".bun", "bin", "claude"),
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		candidates = []string{
			filepath.Join(home, ".local", "bin", "claude.exe"),
			filepath.Join(home, ".local", "bin", "claude.cmd"),
			filepath.Join(home, ".claude", "local", "claude.exe"),
			filepath.Join(appData, "npm", "claude.cmd"),
			filepath.Join(localAppData, "Programs", "claude", "claude.exe"),
		}
	} else {
		candidates = append(candidates, "/opt/homebrew/bin/claude", "/usr/local/bin/claude", "/usr/bin/claude")
	}
	return candidates
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func (a *App) resolveClaude() (string, error) {
	settings, _ := a.store.getSettings()
	if custom := expandHome(settings.ClaudePath); custom != "" {
		if isExecutableFile(custom) {
			return custom, nil
		}
		return "", fmt.Errorf("chemin Claude Code invalide : %s", custom)
	}
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}
	for _, candidate := range claudeCandidates() {
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("Claude Code est introuvable sur cette machine")
}

// claudeCommand prepares an invocation, handling the npm .cmd shim on Windows.
func claudeCommand(ctx context.Context, path string, args []string) *exec.Cmd {
	if runtime.GOOS == "windows" && (strings.HasSuffix(strings.ToLower(path), ".cmd") || strings.HasSuffix(strings.ToLower(path), ".bat")) {
		full := append([]string{"/C", path}, args...)
		return exec.CommandContext(ctx, "cmd", full...)
	}
	return exec.CommandContext(ctx, path, args...)
}

func (a *App) runClaude(prompt string, timeout time.Duration) (string, error) {
	path, err := a.resolveClaude()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := claudeCommand(ctx, path, []string{
		"-p", prompt,
		"--model", claudeModel,
		"--output-format", "text",
	})
	cmd.Env = os.Environ()
	if a.store != nil {
		cmd.Dir = a.store.dataDir
	}
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("Claude Code met trop de temps à répondre, réessaie dans un instant")
	}
	if runErr != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = runErr.Error()
		}
		return "", fmt.Errorf("Claude Code n'a pas pu répondre : %s", clip(detail, 280))
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", errors.New("Claude Code a renvoyé une réponse vide")
	}
	return out, nil
}

func (a *App) claudeStatus() ClaudeStatus {
	status := ClaudeStatus{}
	path, err := a.resolveClaude()
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Path = path
	status.Available = true

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := claudeCommand(ctx, path, []string{"--version"})
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		status.Available = false
		status.Error = "Claude Code a été trouvé mais ne répond pas"
		return status
	}
	version := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(version, '\n'); idx >= 0 {
		version = strings.TrimSpace(version[:idx])
	}
	status.Version = version
	return status
}
