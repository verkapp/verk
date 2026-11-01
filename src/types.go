package src

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
	StartedAt   *int64      `json:"startedAt"`
	TimeSpent   int64       `json:"timeSpent"`
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
	CoachEnabled bool   `json:"coachEnabled"`
	CoachOnStart bool   `json:"coachOnStart"`
	TimeTracking bool   `json:"timeTracking"`
	ShowMomentum bool   `json:"showMomentum"`
	ClaudePath   string `json:"claudePath"`
}

type ClaudeStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Error     string `json:"error"`
}

// SyncStatus is the state of the optional git synchronisation, polled by the
// frontend to refresh the task list after a pull.
type SyncStatus struct {
	Configured   bool   `json:"configured"`
	Remote       string `json:"remote"`
	Branch       string `json:"branch"`
	Enabled      bool   `json:"enabled"`
	Syncing      bool   `json:"syncing"`
	LastSyncAt   int64  `json:"lastSyncAt"`
	LastError    string `json:"lastError"`
	Generation   int64  `json:"generation"`
	GitAvailable bool   `json:"gitAvailable"`
}
