package src

import "strings"

func (s *store) getSettings() (Settings, error) {
	out := Settings{Accent: "lavande", CoachEnabled: true}
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "accent":
			if v != "" {
				out.Accent = v
			}
		case "coach_enabled":
			out.CoachEnabled = v == "1"
		case "coach_on_start":
			out.CoachOnStart = v == "1"
		case "claude_path":
			out.ClaudePath = v
		}
	}
	return out, rows.Err()
}

func (s *store) saveSettings(in Settings) error {
	if in.Accent == "" {
		in.Accent = "lavande"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := map[string]string{
		"accent":         in.Accent,
		"coach_enabled":  boolIntString(in.CoachEnabled),
		"coach_on_start": boolIntString(in.CoachOnStart),
		"claude_path":    strings.TrimSpace(in.ClaudePath),
	}
	for k, v := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
