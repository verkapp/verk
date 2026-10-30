package main

// Task is the shape sent to the Angular frontend. Timestamps are Unix
// milliseconds; nil dates are serialised as null.
type Task struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Done        bool        `json:"done"`
	CreatedAt   int64       `json:"createdAt"`
	DueAt       *int64      `json:"dueAt"`
	CompletedAt *int64      `json:"completedAt"`
	Subtasks    []Subtask   `json:"subtasks"`
	Images      []TaskImage `json:"images"`
}

type Subtask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Position int    `json:"position"`
}

type TaskImage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DataURL   string `json:"dataUrl"`
	CreatedAt int64  `json:"createdAt"`
}

type TaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueAt       *int64 `json:"dueAt"`
}

type Settings struct {
	Accent       string `json:"accent"`
	CoachOnStart bool   `json:"coachOnStart"`
	ClaudePath   string `json:"claudePath"`
}

type ClaudeStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Error     string `json:"error"`
}
