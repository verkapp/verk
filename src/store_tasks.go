package src

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *store) listTasks() ([]Task, error) {
	rows, err := s.db.Query(`SELECT id, title, description, done, created_at, due_at, completed_at, started_at, time_spent FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{}
	index := map[string]int{}
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &done, &t.CreatedAt, &t.DueAt, &t.CompletedAt, &t.StartedAt, &t.TimeSpent); err != nil {
			return nil, err
		}
		t.Done = done == 1
		t.Subtasks = []Subtask{}
		t.Images = []TaskImage{}
		index[t.ID] = len(tasks)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	subRows, err := s.db.Query(`SELECT id, task_id, title, done, position FROM subtasks ORDER BY position ASC, rowid ASC`)
	if err != nil {
		return nil, err
	}
	defer subRows.Close()
	for subRows.Next() {
		var sub Subtask
		var taskID string
		var done int
		if err := subRows.Scan(&sub.ID, &taskID, &sub.Title, &done, &sub.Position); err != nil {
			return nil, err
		}
		sub.Done = done == 1
		if i, ok := index[taskID]; ok {
			tasks[i].Subtasks = append(tasks[i].Subtasks, sub)
		}
	}

	imgRows, err := s.db.Query(`SELECT id, task_id, name, file, created_at FROM images ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer imgRows.Close()
	for imgRows.Next() {
		var id, taskID, name, file string
		var createdAt int64
		if err := imgRows.Scan(&id, &taskID, &name, &file, &createdAt); err != nil {
			return nil, err
		}
		if i, ok := index[taskID]; ok {
			tasks[i].Images = append(tasks[i].Images, TaskImage{
				ID:        id,
				Name:      name,
				DataURL:   s.imageDataURL(file),
				CreatedAt: createdAt,
			})
		}
	}

	sortTasks(tasks)
	return tasks, nil
}

func sortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if a.Done {
			var ac, bc int64
			if a.CompletedAt != nil {
				ac = *a.CompletedAt
			}
			if b.CompletedAt != nil {
				bc = *b.CompletedAt
			}
			return ac > bc
		}
		if (a.DueAt == nil) != (b.DueAt == nil) {
			return a.DueAt != nil
		}
		if a.DueAt != nil && *a.DueAt != *b.DueAt {
			return *a.DueAt < *b.DueAt
		}
		return a.CreatedAt > b.CreatedAt
	})
}

func (s *store) getTask(id string) (Task, error) {
	all, err := s.listTasks()
	if err != nil {
		return Task{}, err
	}
	for _, t := range all {
		if t.ID == id {
			return t, nil
		}
	}
	return Task{}, errNotFound
}

func (s *store) createTask(input TaskInput) (Task, error) {
	title := cleanTitle(input.Title)
	if title == "" {
		return Task{}, errors.New("le titre est obligatoire")
	}
	id := newID()
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`INSERT INTO tasks (id, title, description, done, created_at, due_at, updated_at) VALUES (?, ?, ?, 0, ?, ?, ?)`,
		id, title, strings.TrimSpace(input.Description), now, normalizeDue(input.DueAt), now)
	if err != nil {
		return Task{}, err
	}
	return s.getTask(id)
}

func (s *store) updateTask(id string, input TaskInput) (Task, error) {
	title := cleanTitle(input.Title)
	if title == "" {
		return Task{}, errors.New("le titre est obligatoire")
	}
	res, err := s.db.Exec(`UPDATE tasks SET title = ?, description = ?, due_at = ?, updated_at = ? WHERE id = ?`,
		title, strings.TrimSpace(input.Description), normalizeDue(input.DueAt), time.Now().UnixMilli(), id)
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, errNotFound
	}
	return s.getTask(id)
}

func (s *store) setDone(id string, done bool) (Task, error) {
	var completedAt *int64
	if done {
		now := time.Now().UnixMilli()
		completedAt = &now
	}
	// Finishing (or reopening) a task stops a running timer and banks its time.
	now := time.Now().UnixMilli()
	res, err := s.db.Exec(`UPDATE tasks SET done = ?, completed_at = ?,
		time_spent = time_spent + CASE WHEN started_at IS NULL THEN 0 ELSE MAX(0, ? - started_at) END,
		started_at = NULL, updated_at = ? WHERE id = ?`, boolInt(done), completedAt, now, now, id)
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, errNotFound
	}
	return s.getTask(id)
}

// setStarted starts or stops the timer of an open task. Stopping banks the
// elapsed time into time_spent.
func (s *store) setStarted(id string, started bool) (Task, error) {
	now := time.Now().UnixMilli()
	var res sql.Result
	var err error
	if started {
		res, err = s.db.Exec(`UPDATE tasks SET started_at = COALESCE(started_at, ?), updated_at = ? WHERE id = ? AND done = 0`, now, now, id)
	} else {
		res, err = s.db.Exec(`UPDATE tasks SET
			time_spent = time_spent + CASE WHEN started_at IS NULL THEN 0 ELSE MAX(0, ? - started_at) END,
			started_at = NULL, updated_at = ? WHERE id = ? AND done = 0`, now, now, id)
	}
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, errNotFound
	}
	return s.getTask(id)
}

func (s *store) deleteTask(id string) error {
	files, err := s.imageFiles(id)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	if err := recordTombstone(tx, "task", id, id, time.Now().UnixMilli()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, f := range files {
		_ = os.Remove(filepath.Join(s.imageDir, f))
	}
	return nil
}
