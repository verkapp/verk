package src

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
)

const (
	snapshotVersion    = 2
	tombstoneRetention = 180 * 24 * time.Hour

	tombstoneProfile = "profile"
	tombstoneTask    = "task"
	tombstoneSubtask = "subtask"
	tombstoneImage   = "image"
)

// syncConfig is stored in the local settings table. It is intentionally not
// part of Settings: every machine points at the repository on its own.
type syncConfig struct {
	Remote  string
	Enabled bool
	Branch  string
}

// snapshot is the mergeable representation of the user data committed to git.
// It is flat so that tasks, subtasks and images can be merged independently.
type snapshot struct {
	Version     int                 `json:"version"`
	GeneratedAt int64               `json:"generatedAt"`
	Profiles    []snapshotProfile   `json:"profiles"`
	Tasks       []snapshotTask      `json:"tasks"`
	Subtasks    []snapshotSubtask   `json:"subtasks"`
	Images      []snapshotImage     `json:"images"`
	Tombstones  []snapshotTombstone `json:"tombstones"`
}

type snapshotProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Position  int    `json:"position"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type snapshotTask struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profileId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Position    int64  `json:"position"`
	CreatedAt   int64  `json:"createdAt"`
	DueAt       *int64 `json:"dueAt"`
	CompletedAt *int64 `json:"completedAt"`
	StartedAt   *int64 `json:"startedAt"`
	TimeSpent   int64  `json:"timeSpent"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type snapshotSubtask struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	Title     string `json:"title"`
	Done      bool   `json:"done"`
	Position  int    `json:"position"`
	UpdatedAt int64  `json:"updatedAt"`
}

type snapshotImage struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	Name      string `json:"name"`
	File      string `json:"file"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type snapshotTombstone struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	TaskID    string `json:"taskId,omitempty"`
	DeletedAt int64  `json:"deletedAt"`
}

type snapshotBody struct {
	Version    int                 `json:"version"`
	Profiles   []snapshotProfile   `json:"profiles"`
	Tasks      []snapshotTask      `json:"tasks"`
	Subtasks   []snapshotSubtask   `json:"subtasks"`
	Images     []snapshotImage     `json:"images"`
	Tombstones []snapshotTombstone `json:"tombstones"`
}

func (s snapshot) body() snapshotBody {
	return snapshotBody{
		Version:    s.Version,
		Profiles:   s.Profiles,
		Tasks:      s.Tasks,
		Subtasks:   s.Subtasks,
		Images:     s.Images,
		Tombstones: s.Tombstones,
	}
}

func emptySnapshot() snapshot {
	return snapshot{
		Version:    snapshotVersion,
		Profiles:   []snapshotProfile{},
		Tasks:      []snapshotTask{},
		Subtasks:   []snapshotSubtask{},
		Images:     []snapshotImage{},
		Tombstones: []snapshotTombstone{},
	}
}

func decodeSnapshot(data []byte) (snapshot, error) {
	out := emptySnapshot()
	if len(bytes.TrimSpace(data)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return snapshot{}, fmt.Errorf("instantané de synchronisation illisible : %w", err)
	}
	if out.Version > snapshotVersion {
		return snapshot{}, fmt.Errorf("instantané de synchronisation créé par une version plus récente de Todo (%d)", out.Version)
	}
	if out.Tasks == nil {
		out.Tasks = []snapshotTask{}
	}
	if out.Profiles == nil {
		out.Profiles = []snapshotProfile{}
	}
	if out.Subtasks == nil {
		out.Subtasks = []snapshotSubtask{}
	}
	if out.Images == nil {
		out.Images = []snapshotImage{}
	}
	if out.Tombstones == nil {
		out.Tombstones = []snapshotTombstone{}
	}
	return out, nil
}

// snapshot reads the whole user dataset in a canonical order so two snapshots
// can be compared byte for byte after JSON encoding.
func (s *store) snapshot() (snapshot, error) {
	out := emptySnapshot()
	out.GeneratedAt = time.Now().UnixMilli()

	profileRows, err := s.db.Query(`SELECT id, name, color, position, created_at, updated_at FROM profiles ORDER BY position, created_at, id`)
	if err != nil {
		return snapshot{}, err
	}
	for profileRows.Next() {
		var p snapshotProfile
		if err := profileRows.Scan(&p.ID, &p.Name, &p.Color, &p.Position, &p.CreatedAt, &p.UpdatedAt); err != nil {
			_ = profileRows.Close()
			return snapshot{}, err
		}
		out.Profiles = append(out.Profiles, p)
	}
	if err := profileRows.Close(); err != nil {
		return snapshot{}, err
	}
	if err := profileRows.Err(); err != nil {
		return snapshot{}, err
	}

	rows, err := s.db.Query(`SELECT id, profile_id, title, description, done, position, created_at, due_at, completed_at, started_at, time_spent, updated_at FROM tasks ORDER BY id`)
	if err != nil {
		return snapshot{}, err
	}
	for rows.Next() {
		var t snapshotTask
		var done int
		if err := rows.Scan(&t.ID, &t.ProfileID, &t.Title, &t.Description, &done, &t.Position, &t.CreatedAt, &t.DueAt, &t.CompletedAt, &t.StartedAt, &t.TimeSpent, &t.UpdatedAt); err != nil {
			_ = rows.Close()
			return snapshot{}, err
		}
		t.Done = done == 1
		out.Tasks = append(out.Tasks, t)
	}
	if err := rows.Close(); err != nil {
		return snapshot{}, err
	}
	if err := rows.Err(); err != nil {
		return snapshot{}, err
	}

	subRows, err := s.db.Query(`SELECT id, task_id, title, done, position, updated_at FROM subtasks ORDER BY task_id, position, id`)
	if err != nil {
		return snapshot{}, err
	}
	for subRows.Next() {
		var sub snapshotSubtask
		var done int
		if err := subRows.Scan(&sub.ID, &sub.TaskID, &sub.Title, &done, &sub.Position, &sub.UpdatedAt); err != nil {
			_ = subRows.Close()
			return snapshot{}, err
		}
		sub.Done = done == 1
		out.Subtasks = append(out.Subtasks, sub)
	}
	if err := subRows.Close(); err != nil {
		return snapshot{}, err
	}
	if err := subRows.Err(); err != nil {
		return snapshot{}, err
	}

	imgRows, err := s.db.Query(`SELECT id, task_id, name, file, created_at, updated_at FROM images ORDER BY task_id, created_at, id`)
	if err != nil {
		return snapshot{}, err
	}
	for imgRows.Next() {
		var img snapshotImage
		if err := imgRows.Scan(&img.ID, &img.TaskID, &img.Name, &img.File, &img.CreatedAt, &img.UpdatedAt); err != nil {
			_ = imgRows.Close()
			return snapshot{}, err
		}
		out.Images = append(out.Images, img)
	}
	if err := imgRows.Close(); err != nil {
		return snapshot{}, err
	}
	if err := imgRows.Err(); err != nil {
		return snapshot{}, err
	}

	tombRows, err := s.db.Query(`SELECT kind, id, task_id, deleted_at FROM tombstones ORDER BY kind, id`)
	if err != nil {
		return snapshot{}, err
	}
	for tombRows.Next() {
		var t snapshotTombstone
		if err := tombRows.Scan(&t.Kind, &t.ID, &t.TaskID, &t.DeletedAt); err != nil {
			_ = tombRows.Close()
			return snapshot{}, err
		}
		out.Tombstones = append(out.Tombstones, t)
	}
	if err := tombRows.Close(); err != nil {
		return snapshot{}, err
	}
	if err := tombRows.Err(); err != nil {
		return snapshot{}, err
	}

	return out, nil
}

type tombstoneKey struct {
	kind string
	id   string
}

// mergeSnapshots combines two datasets without touching the database. Entities
// are versioned by updated_at; a deletion wins only when it is newer than the
// entity, so an edit made after a delete resurrects it. Ties are broken on the
// canonical JSON of the entity, which makes both machines reach the same state.
func mergeSnapshots(local, remote snapshot, now int64) snapshot {
	out := emptySnapshot()
	out.GeneratedAt = now
	tombs := mergeTombstones(local.Tombstones, remote.Tombstones, now)

	// A task is considered alive when any of its children or child deletions
	// is newer than the task tombstone. This lets an edit made after a delete
	// resurface the task, without letting child edits overwrite task fields.
	childActivity := map[string]int64{}
	for _, sub := range append(append([]snapshotSubtask{}, local.Subtasks...), remote.Subtasks...) {
		if sub.UpdatedAt > childActivity[sub.TaskID] {
			childActivity[sub.TaskID] = sub.UpdatedAt
		}
	}
	for _, img := range append(append([]snapshotImage{}, local.Images...), remote.Images...) {
		if img.UpdatedAt > childActivity[img.TaskID] {
			childActivity[img.TaskID] = img.UpdatedAt
		}
	}
	for _, tomb := range tombs {
		if tomb.Kind != tombstoneTask && tomb.TaskID != "" && tomb.DeletedAt > childActivity[tomb.TaskID] {
			childActivity[tomb.TaskID] = tomb.DeletedAt
		}
	}

	localProfiles := mapProfiles(local.Profiles)
	remoteProfiles := mapProfiles(remote.Profiles)
	survivingProfiles := map[string]bool{}
	for _, id := range unionKeys(localProfiles, remoteProfiles) {
		l, lok := localProfiles[id]
		r, rok := remoteProfiles[id]
		var win snapshotProfile
		switch {
		case lok && rok:
			win = l
			if remoteWins(l.UpdatedAt, r.UpdatedAt, mustJSON(l), mustJSON(r)) {
				win = r
			}
		case lok:
			win = l
		default:
			win = r
		}
		if t, ok := tombs[tombstoneKey{tombstoneProfile, id}]; ok && win.UpdatedAt <= t.DeletedAt {
			continue
		}
		out.Profiles = append(out.Profiles, win)
		survivingProfiles[id] = true
	}
	// Tasks always need a container. If a profile was deleted on another
	// machine, its tasks fall back to the first remaining profile.
	fallbackProfile := ""
	fallbackPosition := 0
	for _, p := range out.Profiles {
		if fallbackProfile == "" || p.Position < fallbackPosition {
			fallbackProfile = p.ID
			fallbackPosition = p.Position
		}
	}

	localTasks := mapTasks(local.Tasks)
	remoteTasks := mapTasks(remote.Tasks)
	surviving := map[string]bool{}
	for _, id := range unionKeys(localTasks, remoteTasks) {
		l, lok := localTasks[id]
		r, rok := remoteTasks[id]
		var win snapshotTask
		switch {
		case lok && rok:
			win = l
			if remoteWins(l.UpdatedAt, r.UpdatedAt, mustJSON(l), mustJSON(r)) {
				win = r
			}
		case lok:
			win = l
		default:
			win = r
		}
		if t, ok := tombs[tombstoneKey{tombstoneTask, id}]; ok {
			aliveAt := win.UpdatedAt
			if childActivity[id] > aliveAt {
				aliveAt = childActivity[id]
			}
			if aliveAt <= t.DeletedAt {
				continue
			}
		}
		if !survivingProfiles[win.ProfileID] {
			win.ProfileID = fallbackProfile
		}
		out.Tasks = append(out.Tasks, win)
		surviving[id] = true
	}

	localSubtasks := mapSubtasks(local.Subtasks)
	remoteSubtasks := mapSubtasks(remote.Subtasks)
	for _, id := range unionKeys(localSubtasks, remoteSubtasks) {
		l, lok := localSubtasks[id]
		r, rok := remoteSubtasks[id]
		var win snapshotSubtask
		switch {
		case lok && rok:
			win = l
			if remoteWins(l.UpdatedAt, r.UpdatedAt, mustJSON(l), mustJSON(r)) {
				win = r
			}
		case lok:
			win = l
		default:
			win = r
		}
		if !surviving[win.TaskID] {
			continue
		}
		if t, ok := tombs[tombstoneKey{tombstoneSubtask, id}]; ok && win.UpdatedAt <= t.DeletedAt {
			continue
		}
		out.Subtasks = append(out.Subtasks, win)
	}

	localImages := mapImages(local.Images)
	remoteImages := mapImages(remote.Images)
	for _, id := range unionKeys(localImages, remoteImages) {
		l, lok := localImages[id]
		r, rok := remoteImages[id]
		var win snapshotImage
		switch {
		case lok && rok:
			win = l
			if remoteWins(l.UpdatedAt, r.UpdatedAt, mustJSON(l), mustJSON(r)) {
				win = r
			}
		case lok:
			win = l
		default:
			win = r
		}
		if !surviving[win.TaskID] {
			continue
		}
		if t, ok := tombs[tombstoneKey{tombstoneImage, id}]; ok && win.UpdatedAt <= t.DeletedAt {
			continue
		}
		out.Images = append(out.Images, win)
	}

	for _, t := range tombs {
		out.Tombstones = append(out.Tombstones, t)
	}

	sort.Slice(out.Profiles, func(i, j int) bool {
		a, b := out.Profiles[i], out.Profiles[j]
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.ID < b.ID
	})
	sort.Slice(out.Tasks, func(i, j int) bool { return out.Tasks[i].ID < out.Tasks[j].ID })
	sort.Slice(out.Subtasks, func(i, j int) bool {
		a, b := out.Subtasks[i], out.Subtasks[j]
		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		return a.ID < b.ID
	})
	sort.Slice(out.Images, func(i, j int) bool {
		a, b := out.Images[i], out.Images[j]
		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.ID < b.ID
	})
	sort.Slice(out.Tombstones, func(i, j int) bool {
		a, b := out.Tombstones[i], out.Tombstones[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	return out
}

func mergeTombstones(local, remote []snapshotTombstone, now int64) map[tombstoneKey]snapshotTombstone {
	cutoff := now - tombstoneRetention.Milliseconds()
	out := map[tombstoneKey]snapshotTombstone{}
	add := func(t snapshotTombstone) {
		if t.Kind == "" || t.ID == "" || t.DeletedAt < cutoff {
			return
		}
		key := tombstoneKey{t.Kind, t.ID}
		if existing, ok := out[key]; !ok || t.DeletedAt > existing.DeletedAt {
			out[key] = t
		}
	}
	for _, t := range local {
		add(t)
	}
	for _, t := range remote {
		add(t)
	}
	return out
}

func remoteWins(localAt, remoteAt int64, localKey, remoteKey string) bool {
	if localAt != remoteAt {
		return remoteAt > localAt
	}
	return remoteKey > localKey
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func mapTasks(items []snapshotTask) map[string]snapshotTask {
	out := make(map[string]snapshotTask, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

func mapProfiles(items []snapshotProfile) map[string]snapshotProfile {
	out := make(map[string]snapshotProfile, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

func mapSubtasks(items []snapshotSubtask) map[string]snapshotSubtask {
	out := make(map[string]snapshotSubtask, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

func mapImages(items []snapshotImage) map[string]snapshotImage {
	out := make(map[string]snapshotImage, len(items))
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

func unionKeys[T any](a, b map[string]T) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		seen[key] = struct{}{}
	}
	for key := range b {
		seen[key] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// mergeSnapshot merges the remote snapshot into the local database and returns
// the merged result plus whether the local data changed.
func (s *store) mergeSnapshot(remote snapshot) (snapshot, bool, error) {
	local, err := s.snapshot()
	if err != nil {
		return snapshot{}, false, err
	}
	merged := mergeSnapshots(local, remote, time.Now().UnixMilli())

	// A timer running on this machine stays local: importing a remote
	// started_at would otherwise start phantom timers everywhere.
	running := map[string]int64{}
	for _, t := range local.Tasks {
		if t.StartedAt != nil && !t.Done {
			running[t.ID] = *t.StartedAt
		}
	}
	for i := range merged.Tasks {
		if at, ok := running[merged.Tasks[i].ID]; ok {
			value := at
			merged.Tasks[i].StartedAt = &value
		} else {
			merged.Tasks[i].StartedAt = nil
		}
	}

	changed := !snapshotsEqual(local, merged)
	if changed {
		if err := s.replaceSnapshot(merged); err != nil {
			return snapshot{}, false, err
		}
	}
	return merged, changed, nil
}

func snapshotsEqual(a, b snapshot) bool {
	left, err := json.Marshal(a.body())
	if err != nil {
		return false
	}
	right, err := json.Marshal(b.body())
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}

func (s *store) replaceSnapshot(snap snapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range []string{`DELETE FROM subtasks`, `DELETE FROM images`, `DELETE FROM tasks`, `DELETE FROM profiles`, `DELETE FROM tombstones`} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	for _, p := range snap.Profiles {
		if _, err := tx.Exec(`INSERT INTO profiles (id, name, color, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Color, p.Position, p.CreatedAt, p.UpdatedAt); err != nil {
			return err
		}
	}
	for _, t := range snap.Tasks {
		if _, err := tx.Exec(`INSERT INTO tasks (id, profile_id, title, description, done, position, created_at, due_at, completed_at, started_at, time_spent, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.ProfileID, t.Title, t.Description, boolInt(t.Done), t.Position, t.CreatedAt, t.DueAt, t.CompletedAt, t.StartedAt, t.TimeSpent, t.UpdatedAt); err != nil {
			return err
		}
	}
	for _, sub := range snap.Subtasks {
		if _, err := tx.Exec(`INSERT INTO subtasks (id, task_id, title, done, position, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			sub.ID, sub.TaskID, sub.Title, boolInt(sub.Done), sub.Position, sub.UpdatedAt); err != nil {
			return err
		}
	}
	for _, img := range snap.Images {
		if _, err := tx.Exec(`INSERT INTO images (id, task_id, name, file, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			img.ID, img.TaskID, img.Name, img.File, img.CreatedAt, img.UpdatedAt); err != nil {
			return err
		}
	}
	for _, t := range snap.Tombstones {
		if _, err := tx.Exec(`INSERT INTO tombstones (kind, id, task_id, deleted_at) VALUES (?, ?, ?, ?)`,
			t.Kind, t.ID, t.TaskID, t.DeletedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func recordTombstone(ex sqlExecer, kind, id, taskID string, at int64) error {
	_, err := ex.Exec(`INSERT INTO tombstones (kind, id, task_id, deleted_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(kind, id) DO UPDATE SET task_id = excluded.task_id, deleted_at = excluded.deleted_at`,
		kind, id, taskID, at)
	return err
}

func (s *store) syncConfig() (syncConfig, error) {
	out := syncConfig{Enabled: true, Branch: "main"}
	rows, err := s.db.Query(`SELECT key, value FROM settings WHERE key IN ('sync_remote', 'sync_enabled', 'sync_branch')`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return out, err
		}
		switch key {
		case "sync_remote":
			out.Remote = value
		case "sync_enabled":
			out.Enabled = value == "1"
		case "sync_branch":
			if value != "" {
				out.Branch = value
			}
		}
	}
	return out, rows.Err()
}

func (s *store) saveSyncConfig(cfg syncConfig) error {
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	values := map[string]string{
		"sync_remote":  cfg.Remote,
		"sync_enabled": boolIntString(cfg.Enabled),
		"sync_branch":  cfg.Branch,
	}
	return s.saveSettingValues(values)
}

func (s *store) syncMeta() (lastAt int64, lastError string, err error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings WHERE key IN ('sync_last_at', 'sync_last_error')`)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return 0, "", err
		}
		switch key {
		case "sync_last_at":
			if value != "" {
				parsed, parseErr := strconv.ParseInt(value, 10, 64)
				if parseErr == nil {
					lastAt = parsed
				}
			}
		case "sync_last_error":
			lastError = value
		}
	}
	return lastAt, lastError, rows.Err()
}

func (s *store) saveSyncMeta(lastAt int64, lastError string) error {
	return s.saveSettingValues(map[string]string{
		"sync_last_at":    fmt.Sprintf("%d", lastAt),
		"sync_last_error": lastError,
	})
}

func (s *store) saveSettingValues(values map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for key, value := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}
