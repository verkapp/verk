package src

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

var profileColors = []string{"#7b6cf0", "#4aa985", "#559fe0", "#ea8a62", "#dd7ba2", "#d3a640"}

const maxProfileName = 40

func (s *store) listProfiles() ([]Profile, error) {
	rows, err := s.db.Query(`SELECT id, name, color, position, created_at, auto_defer_at FROM profiles ORDER BY position ASC, created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := []Profile{}
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.Color, &p.Position, &p.CreatedAt, &p.AutoDeferAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *store) getProfile(id string) (Profile, error) {
	var p Profile
	err := s.db.QueryRow(`SELECT id, name, color, position, created_at, auto_defer_at FROM profiles WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Color, &p.Position, &p.CreatedAt, &p.AutoDeferAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, errNotFound
	}
	return p, err
}

func (s *store) createProfile(name string) (Profile, error) {
	name = clip(strings.TrimSpace(name), maxProfileName)
	if name == "" {
		return Profile{}, errors.New("le nom du profil est obligatoire")
	}
	var count, position int
	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(position), -1) + 1 FROM profiles`).Scan(&count, &position); err != nil {
		return Profile{}, err
	}
	color := profileColors[count%len(profileColors)]
	id := newID()
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`INSERT INTO profiles (id, name, color, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, name, color, position, now, now); err != nil {
		return Profile{}, err
	}
	return s.getProfile(id)
}

func (s *store) updateProfile(id, name, color string) (Profile, error) {
	current, err := s.getProfile(id)
	if err != nil {
		return Profile{}, err
	}
	name = clip(strings.TrimSpace(name), maxProfileName)
	if name == "" {
		return Profile{}, errors.New("le nom du profil est obligatoire")
	}
	color = strings.TrimSpace(color)
	if color == "" {
		color = current.Color
	}
	if len(color) > 32 {
		color = current.Color
	}
	if _, err := s.db.Exec(`UPDATE profiles SET name = ?, color = ?, updated_at = ? WHERE id = ?`,
		name, color, time.Now().UnixMilli(), id); err != nil {
		return Profile{}, err
	}
	return s.getProfile(id)
}

// setProfileAutoDefer sets (or clears with "") the auto-defer time. The local
// last-run stamp is reset to now, so the new time only applies from its next
// occurrence: changing the setting never moves tasks on the spot.
func (s *store) setProfileAutoDefer(id, at string) (Profile, error) {
	if _, err := s.getProfile(id); err != nil {
		return Profile{}, err
	}
	normalised := ""
	if strings.TrimSpace(at) != "" {
		_, _, clock, err := parseClock(at)
		if err != nil {
			return Profile{}, err
		}
		normalised = clock
	}
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`UPDATE profiles SET auto_defer_at = ?, updated_at = ? WHERE id = ?`, normalised, now, id); err != nil {
		return Profile{}, err
	}
	if err := s.saveSettingValues(map[string]string{autoDeferKey(id): strconv.FormatInt(now, 10)}); err != nil {
		return Profile{}, err
	}
	return s.getProfile(id)
}

// deleteProfile removes a profile and moves its tasks to the next remaining
// profile, so no task is ever left without a container.
func (s *store) deleteProfile(id string) error {
	profiles, err := s.listProfiles()
	if err != nil {
		return err
	}
	if len(profiles) <= 1 {
		return errors.New("il faut conserver au moins un profil")
	}
	var replacement string
	for _, p := range profiles {
		if p.ID != id {
			replacement = p.ID
			break
		}
	}
	if replacement == "" {
		return errNotFound
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var next int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM tasks WHERE profile_id = ?`, replacement).Scan(&next); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT id FROM tasks WHERE profile_id = ? ORDER BY position ASC, created_at ASC, id ASC`, id)
	if err != nil {
		return err
	}
	moved := []string{}
	for rows.Next() {
		var taskID string
		if err := rows.Scan(&taskID); err != nil {
			_ = rows.Close()
			return err
		}
		moved = append(moved, taskID)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, taskID := range moved {
		if _, err := tx.Exec(`UPDATE tasks SET profile_id = ?, position = ?, updated_at = ? WHERE id = ?`,
			replacement, next, now, taskID); err != nil {
			return err
		}
		next++
	}
	res, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	if err := recordTombstone(tx, tombstoneProfile, id, "", now); err != nil {
		return err
	}
	return tx.Commit()
}
