package main

import (
	"bytes"
	"context"
	"encoding/json"
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

func (a *App) breakDownTask(taskID string) ([]string, error) {
	task, err := a.store.getTask(taskID)
	if err != nil {
		return nil, err
	}
	description := strings.TrimSpace(task.Description)
	if description == "" {
		description = "(aucune description)"
	}
	prompt := fmt.Sprintf(`Tu aides à planifier une tâche dans une application de todo.
Décompose la tâche ci-dessous en 3 à 6 sous-tâches concrètes et actionnables.

Règles :
- chaque sous-tâche commence par un verbe à l'infinitif ;
- 70 caractères maximum par sous-tâche ;
- pas de numérotation, pas de ponctuation finale inutile ;
- réponds UNIQUEMENT par un tableau JSON de chaînes, sans backticks ni commentaire.

Titre : %s
Description : %s`, task.Title, clip(description, 3000))

	out, err := a.runClaude(prompt, 90*time.Second)
	if err != nil {
		return nil, err
	}
	return parseSuggestions(out)
}

// parseSuggestions tolerates models wrapping JSON in prose or code fences.
func parseSuggestions(raw string) ([]string, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if start, end := strings.Index(cleaned, "["), strings.LastIndex(cleaned, "]"); start >= 0 && end > start {
		var items []string
		if err := json.Unmarshal([]byte(cleaned[start:end+1]), &items); err == nil {
			out := dedupeSuggestions(items)
			if len(out) > 0 {
				return out, nil
			}
		}
	}

	// Fallback: one suggestion per line.
	var out []string
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-*•0123456789.) \t")
		line = strings.Trim(line, `"'`)
		if line != "" && !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "[") {
			out = append(out, line)
		}
	}
	out = dedupeSuggestions(out)
	if len(out) == 0 {
		return nil, errors.New("Claude n'a pas renvoyé de sous-tâches exploitables")
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return out, nil
}

func dedupeSuggestions(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		item = strings.TrimSpace(strings.Trim(item, `"'`))
		item = strings.TrimLeft(item, "-*• ")
		item = clip(item, 120)
		key := strings.ToLower(item)
		if item == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
		if len(out) == 8 {
			break
		}
	}
	return out
}

func (a *App) coachNudge() (string, error) {
	tasks, err := a.store.listTasks()
	if err != nil {
		return "", err
	}
	now := time.Now()
	var open []Task
	var doneToday int
	var overdue int
	for _, t := range tasks {
		if t.Done {
			if t.CompletedAt != nil && sameLocalDay(time.UnixMilli(*t.CompletedAt), now) {
				doneToday++
			}
			continue
		}
		open = append(open, t)
		if t.DueAt != nil && *t.DueAt < now.UnixMilli() {
			overdue++
		}
	}
	if len(open) == 0 {
		return "Ta liste est vide : profite de ce moment pour souffler, ou note une petite idée pour plus tard.", nil
	}

	var lines []string
	for i, t := range open {
		if i >= 12 {
			lines = append(lines, fmt.Sprintf("… et %d autre(s)", len(open)-12))
			break
		}
		age := int(now.Sub(time.UnixMilli(t.CreatedAt)).Hours() / 24)
		due := "sans échéance"
		if t.DueAt != nil {
			dueAt := time.UnixMilli(*t.DueAt)
			if dueAt.Before(now) {
				due = "en retard depuis le " + dueAt.Format("02/01")
			} else {
				due = "pour le " + dueAt.Format("02/01 à 15:04")
			}
		}
		lines = append(lines, fmt.Sprintf("- %s (créée il y a %d j, %s)", t.Title, age, due))
	}

	weekday := []string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}[int(now.Weekday())]
	prompt := fmt.Sprintf(`Tu es un coach doux et concret intégré à une application de todo.
Nous sommes %s %s. État de la liste :
- %d tâche(s) ouverte(s), dont %d en retard ;
- %d tâche(s) terminée(s) aujourd'hui.

Tâches ouvertes :
%s

Écris un rappel en français, au tutoiement, en 2 phrases maximum (40 mots max).
Le ton est bienveillant et jamais culpabilisant. Termine par UNE seule action très simple et immédiate.
Pas de liste, pas de titre, pas d'emoji, pas de guillemets. Réponds uniquement par le texte.`,
		weekday, now.Format("02/01/2006"), len(open), overdue, doneToday, strings.Join(lines, "\n"))

	out, err := a.runClaude(prompt, 90*time.Second)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	out = strings.Trim(out, `"`)
	return out, nil
}

func sameLocalDay(a, b time.Time) bool {
	ay, am, ad := a.Local().Date()
	by, bm, bd := b.Local().Date()
	return ay == by && am == bm && ad == bd
}
