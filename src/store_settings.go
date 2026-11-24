package src

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
)

func (s *store) getSettings() (Settings, error) {
	out := Settings{Accent: "lavande", Theme: themeSystem, CoachEnabled: true, TimeTracking: true, ShowMomentum: true, Notifications: true, AutoUpdate: true, ClaudeEnabled: true}
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
		case "theme":
			out.Theme = normalizeTheme(v)
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
		"theme":          normalizeTheme(in.Theme),
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

// settingInt reads an integer stored in the local settings table.
func (s *store) settingInt(key string) (int64, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, nil
	}
	return parsed, true, nil
}

// Appearance preferences: follow the operating system, or force a theme.
const (
	themeSystem = "system"
	themeLight  = "light"
	themeDark   = "dark"
)

// normalizeTheme maps unknown or empty values to themeSystem.
func normalizeTheme(value string) string {
	switch value {
	case themeLight, themeDark:
		return value
	default:
		return themeSystem
	}
}
