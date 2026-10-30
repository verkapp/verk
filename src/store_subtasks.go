package src

import (
	"database/sql"
	"errors"
	"strings"
)

func (s *store) addSubtask(taskID, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("la sous-tâche est vide")
	}
	var pos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM subtasks WHERE task_id = ?`, taskID).Scan(&pos)
	if _, err := s.db.Exec(`INSERT INTO subtasks (id, task_id, title, done, position) VALUES (?, ?, ?, 0, ?)`,
		newID(), taskID, clip(title, 240), pos); err != nil {
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
	if _, err := s.db.Exec(`UPDATE subtasks SET title = ? WHERE id = ?`, clip(title, 240), id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) setSubtaskDone(id string, done bool) (Task, error) {
	taskID, err := s.subtaskTaskID(id)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.db.Exec(`UPDATE subtasks SET done = ? WHERE id = ?`, boolInt(done), id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) deleteSubtask(id string) (Task, error) {
	taskID, err := s.subtaskTaskID(id)
	if err != nil {
		return Task{}, err
	}
	if _, err := s.db.Exec(`DELETE FROM subtasks WHERE id = ?`, id); err != nil {
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
