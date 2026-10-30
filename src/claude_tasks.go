package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
