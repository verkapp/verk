package src

import (
	"database/sql"
	"errors"
	"time"
)

// reorderTasks applies a manual order to the given tasks. The caller sends the
// complete list of open tasks for a profile, in display order.
func (s *store) reorderTasks(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seen := make(map[string]bool, len(ids))
	position := int64(0)
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		res, err := tx.Exec(`UPDATE tasks SET position = ?, updated_at = ? WHERE id = ?`, position, now, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errNotFound
		}
		position++
	}
	return tx.Commit()
}

// reorderSubtasks applies a manual order to the subtasks of one task.
func (s *store) reorderSubtasks(taskID string, ids []string) (Task, error) {
	var exists int
	if err := s.db.QueryRow(`SELECT 1 FROM tasks WHERE id = ?`, taskID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()
	seen := make(map[string]bool, len(ids))
	position := 0
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		res, err := tx.Exec(`UPDATE subtasks SET position = ?, updated_at = ? WHERE id = ? AND task_id = ?`,
			position, now, id, taskID)
		if err != nil {
			return Task{}, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Task{}, errNotFound
		}
		position++
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}
