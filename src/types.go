package src

// Task is the shape sent to the Angular frontend. Timestamps are Unix
// milliseconds; nil dates are serialised as null.
type Task struct {
	ID          string      `json:"id"`
	ProfileID   string      `json:"profileId"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Done        bool        `json:"done"`
	Position    int64       `json:"position"`
	CreatedAt   int64       `json:"createdAt"`
	DueAt       *int64      `json:"dueAt"`
	CompletedAt *int64      `json:"completedAt"`
	StartedAt   *int64      `json:"startedAt"`
	TimeSpent   int64       `json:"timeSpent"`
	Subtasks    []Subtask   `json:"subtasks"`
	Images      []TaskImage `json:"images"`
}

// Profile is a container for tasks (Travail, Perso, …). Positions only order
// the profile list; tasks are ordered independently inside each profile.
type Profile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Position  int    `json:"position"`
	CreatedAt int64  `json:"createdAt"`
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
	ProfileID   string `json:"profileId"`
}

type Settings struct {
	Accent        string `json:"accent"`
	CoachEnabled  bool   `json:"coachEnabled"`
	CoachOnStart  bool   `json:"coachOnStart"`
	TimeTracking  bool   `json:"timeTracking"`
	ShowMomentum  bool   `json:"showMomentum"`
	Notifications bool   `json:"notifications"`
	ClaudePath    string `json:"claudePath"`
}

// NotificationStatus describes whether desktop notifications can be shown.
type NotificationStatus struct {
	Available  bool   `json:"available"`
	Authorized bool   `json:"authorized"`
	Error      string `json:"error"`
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
