package src

import "strings"

func (s *store) getSettings() (Settings, error) {
	out := Settings{Accent: "lavande", CoachEnabled: true, TimeTracking: true, ShowMomentum: true, Notifications: true, AutoUpdate: true, ClaudeEnabled: true}
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
		case "time_tracking":
			out.TimeTracking = v == "1"
		case "show_momentum":
			out.ShowMomentum = v == "1"
		case "notifications":
			out.Notifications = v == "1"
		case "auto_update":
			out.AutoUpdate = v == "1"
		case "claude_enabled":
			out.ClaudeEnabled = v == "1"
		case "close_to_tray":
			out.CloseToTray = v == "1"
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
		"time_tracking":  boolIntString(in.TimeTracking),
		"show_momentum":  boolIntString(in.ShowMomentum),
		"notifications":  boolIntString(in.Notifications),
		"auto_update":    boolIntString(in.AutoUpdate),
		"claude_enabled": boolIntString(in.ClaudeEnabled),
		"claude_path":    strings.TrimSpace(in.ClaudePath),
		"close_to_tray":  boolIntString(in.CloseToTray),
	}
	for k, v := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
