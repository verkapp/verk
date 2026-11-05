package src

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *store) addSubtask(taskID, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("la sous-tâche est vide")
	}
	var pos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM subtasks WHERE task_id = ?`, taskID).Scan(&pos)
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`INSERT INTO subtasks (id, task_id, title, done, position, updated_at) VALUES (?, ?, ?, 0, ?, ?)`,
		newID(), taskID, clip(title, 240), pos, now); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) addSubtasks(taskID string, titles []string) (Task, error) {
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		if _, err := s.addSubtask(taskID, title); err != nil {
			return Task{}, err
		}
	}
	return s.getTask(taskID)
}

func (s *store) updateSubtask(id, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("la sous-tâche est vide")
	}
	taskID, err := s.subtaskTaskID(id)
	if err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`UPDATE subtasks SET title = ?, updated_at = ? WHERE id = ?`, clip(title, 240), now, id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) setSubtaskDone(id string, done bool) (Task, error) {
	taskID, err := s.subtaskTaskID(id)
	if err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`UPDATE subtasks SET done = ?, updated_at = ? WHERE id = ?`, boolInt(done), now, id); err != nil {
		return Task{}, err
	}
	// Finishing the last open subtask finishes the task too, keeping the two
	// consistent in both directions.
	if done {
		complete, err := s.allSubtasksDone(taskID)
		if err != nil {
			return Task{}, err
		}
		if complete {
			return s.setDone(taskID, true)
		}
	}
	return s.getTask(taskID)
}

// allSubtasksDone reports whether a task has subtasks and all of them are done.
func (s *store) allSubtasksDone(taskID string) (bool, error) {
	var total, done int
	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(done), 0) FROM subtasks WHERE task_id = ?`, taskID).Scan(&total, &done); err != nil {
		return false, err
	}
	return total > 0 && total == done, nil
}

func (s *store) deleteSubtask(id string) (Task, error) {
	taskID, err := s.subtaskTaskID(id)
	if err != nil {
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM subtasks WHERE id = ?`, id); err != nil {
		return Task{}, err
	}
	if err := recordTombstone(tx, "subtask", id, taskID, now); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) subtaskTaskID(id string) (string, error) {
	var taskID string
	if err := s.db.QueryRow(`SELECT task_id FROM subtasks WHERE id = ?`, id).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errNotFound
		}
		return "", err
	}
	return taskID, nil
}
