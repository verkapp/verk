package src

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func (a *App) coachNudge() (string, error) {
	settings, err := a.store.getSettings()
	if err != nil {
		return "", err
	}
	if !settings.CoachEnabled {
		return "", errors.New("le coach est désactivé")
	}
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
