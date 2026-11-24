package src

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Layout of the sync repository since format 3: one folder per profile and
// one JSON file per task, so a commit only touches the tasks that changed.
//
//	profiles/<profile-id>/profile.json
//	profiles/<profile-id>/tasks/<task-id>.json   task + its subtasks + its images
//	tombstones.json                              deletions, kept six months
//	images/                                      attachment files (unchanged)
//	todo.json                                    fence for older versions
//
// Folders are named after IDs, not names, so renaming a profile moves
// nothing. The profileId field inside a task file is authoritative; the
// folder only follows it.
const (
	syncFormatVersion  = 3
	profilesDirName    = "profiles"
	profileFileName    = "profile.json"
	tasksDirName       = "tasks"
	tombstonesFileName = "tombstones.json"
)

// layoutFence replaces todo.json once a repository uses the folder layout.
// Versions up to 2.0 refuse a snapshot whose version is above 2, so they stop
// with « créé par une version plus récente » instead of pushing an old,
// complete todo.json over the new layout.
type layoutFence struct {
	Version int    `json:"version"`
	Layout  string `json:"layout"`
	Note    string `json:"note"`
}

func newLayoutFence() layoutFence {
	return layoutFence{
		Version: syncFormatVersion,
		Layout:  "profiles/<profil>/tasks/<tâche>.json",
		Note:    "Les tâches sont rangées dans profiles/. Mets Verk à jour sur cet appareil pour synchroniser.",
	}
}

// taskFile is the content of profiles/<profile-id>/tasks/<task-id>.json.
type taskFile struct {
	snapshotTask
	Subtasks []taskFileSubtask `json:"subtasks"`
	Images   []taskFileImage   `json:"images"`
}

type taskFileSubtask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Done      bool   `json:"done"`
	Position  int    `json:"position"`
	UpdatedAt int64  `json:"updatedAt"`
}

type taskFileImage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	File      string `json:"file"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type tombstonesFile struct {
	Tombstones []snapshotTombstone `json:"tombstones"`
}

// repoSnapshot is what a sync repository holds: the dataset, plus whether it
// still contained a pre-format-3 todo.json that needs migrating.
type repoSnapshot struct {
	snapshot
	legacy bool
}

// readRepoSnapshot loads the dataset of a sync repository, whatever its
// format. A legacy todo.json (format 2) is merged with the folder layout when
// both are present: an older device may have pushed one last time before
// meeting the fence, and none of its changes must be lost.
func readRepoSnapshot(dir string, now int64) (repoSnapshot, error) {
	tree, err := readSnapshotTree(dir)
	if err != nil {
		return repoSnapshot{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, snapshotFileName))
	if errors.Is(err, os.ErrNotExist) {
		return repoSnapshot{snapshot: tree}, nil
	}
	if err != nil {
		return repoSnapshot{}, err
	}
	var header struct {
		Version int `json:"version"`
	}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &header); err != nil {
			return repoSnapshot{}, fmt.Errorf("instantané de synchronisation illisible : %w", err)
		}
	}
	switch {
	case header.Version > syncFormatVersion:
		return repoSnapshot{}, fmt.Errorf("instantané de synchronisation créé par une version plus récente de Verk (%d)", header.Version)
	case header.Version == syncFormatVersion:
		return repoSnapshot{snapshot: tree}, nil
	}
	legacy, err := decodeSnapshot(data)
	if err != nil {
		return repoSnapshot{}, err
	}
	return repoSnapshot{snapshot: mergeSnapshots(tree, legacy, now), legacy: true}, nil
}

// readSnapshotTree reads profiles/ and tombstones.json. A missing folder is
// an empty dataset, as for a repository that was never migrated.
func readSnapshotTree(dir string) (snapshot, error) {
	out := emptySnapshot()

	data, err := os.ReadFile(filepath.Join(dir, tombstonesFileName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return snapshot{}, err
	}
	if err == nil {
		var file tombstonesFile
		if err := json.Unmarshal(data, &file); err != nil {
			return snapshot{}, fmt.Errorf("%s illisible : %w", tombstonesFileName, err)
		}
		out.Tombstones = append(out.Tombstones, file.Tombstones...)
	}

	profileDirs, err := os.ReadDir(filepath.Join(dir, profilesDirName))
	if errors.Is(err, os.ErrNotExist) {
		return sortSnapshot(out), nil
	}
	if err != nil {
		return snapshot{}, err
	}
	for _, entry := range profileDirs {
		if !entry.IsDir() {
			continue
		}
		profileDir := filepath.Join(dir, profilesDirName, entry.Name())
		data, err := os.ReadFile(filepath.Join(profileDir, profileFileName))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return snapshot{}, err
		}
		if err == nil {
			var profile snapshotProfile
			if err := json.Unmarshal(data, &profile); err != nil {
				return snapshot{}, fmt.Errorf("profil %s illisible : %w", entry.Name(), err)
			}
			out.Profiles = append(out.Profiles, profile)
		}

		taskEntries, err := os.ReadDir(filepath.Join(profileDir, tasksDirName))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return snapshot{}, err
		}
		for _, taskEntry := range taskEntries {
			if taskEntry.IsDir() || filepath.Ext(taskEntry.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(profileDir, tasksDirName, taskEntry.Name()))
			if err != nil {
				return snapshot{}, err
			}
			var file taskFile
			if err := json.Unmarshal(data, &file); err != nil {
				return snapshot{}, fmt.Errorf("tâche %s illisible : %w", taskEntry.Name(), err)
			}
			out.Tasks = append(out.Tasks, file.snapshotTask)
			for _, sub := range file.Subtasks {
				out.Subtasks = append(out.Subtasks, snapshotSubtask{
					ID: sub.ID, TaskID: file.ID, Title: sub.Title, Done: sub.Done, Position: sub.Position, UpdatedAt: sub.UpdatedAt,
				})
			}
			for _, img := range file.Images {
				out.Images = append(out.Images, snapshotImage{
					ID: img.ID, TaskID: file.ID, Name: img.Name, File: img.File, CreatedAt: img.CreatedAt, UpdatedAt: img.UpdatedAt,
				})
			}
		}
	}
	return sortSnapshot(out), nil
}

// sortSnapshot puts a snapshot read from files in the canonical order of
// store.snapshot, so equal datasets compare equal.
func sortSnapshot(s snapshot) snapshot {
	sort.SliceStable(s.Profiles, func(i, j int) bool {
		a, b := s.Profiles[i], s.Profiles[j]
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.ID < b.ID
	})
	sort.SliceStable(s.Tasks, func(i, j int) bool { return s.Tasks[i].ID < s.Tasks[j].ID })
	sort.SliceStable(s.Subtasks, func(i, j int) bool {
		a, b := s.Subtasks[i], s.Subtasks[j]
		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		return a.ID < b.ID
	})
	sort.SliceStable(s.Images, func(i, j int) bool {
		a, b := s.Images[i], s.Images[j]
		if a.TaskID != b.TaskID {
			return a.TaskID < b.TaskID
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.ID < b.ID
	})
	sort.SliceStable(s.Tombstones, func(i, j int) bool {
		a, b := s.Tombstones[i], s.Tombstones[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	return s
}

// writeSnapshotTree writes the dataset in the folder layout and the fence.
// Files whose content is unchanged are left alone, and files of tasks or
// profiles that no longer exist are removed.
func writeSnapshotTree(dir string, snap snapshot) error {
	snap = sortSnapshot(snap)
	want := map[string][]byte{}
	add := func(rel string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		want[filepath.Join(dir, rel)] = append(data, '\n')
		return nil
	}

	if err := add(snapshotFileName, newLayoutFence()); err != nil {
		return err
	}
	if err := add(tombstonesFileName, tombstonesFile{Tombstones: snap.Tombstones}); err != nil {
		return err
	}
	for _, profile := range snap.Profiles {
		if err := add(filepath.Join(profilesDirName, safeName(profile.ID), profileFileName), profile); err != nil {
			return err
		}
	}

	subtasks := map[string][]taskFileSubtask{}
	for _, sub := range snap.Subtasks {
		subtasks[sub.TaskID] = append(subtasks[sub.TaskID], taskFileSubtask{
			ID: sub.ID, Title: sub.Title, Done: sub.Done, Position: sub.Position, UpdatedAt: sub.UpdatedAt,
		})
	}
	images := map[string][]taskFileImage{}
	for _, img := range snap.Images {
		images[img.TaskID] = append(images[img.TaskID], taskFileImage{
			ID: img.ID, Name: img.Name, File: img.File, CreatedAt: img.CreatedAt, UpdatedAt: img.UpdatedAt,
		})
	}
	for _, task := range snap.Tasks {
		file := taskFile{
			snapshotTask: task,
			Subtasks:     subtasks[task.ID],
			Images:       images[task.ID],
		}
		if file.Subtasks == nil {
			file.Subtasks = []taskFileSubtask{}
		}
		if file.Images == nil {
			file.Images = []taskFileImage{}
		}
		rel := filepath.Join(profilesDirName, safeName(task.ProfileID), tasksDirName, safeName(task.ID)+".json")
		if err := add(rel, file); err != nil {
			return err
		}
	}

	for path, data := range want {
		if err := writeFileIfChanged(path, data); err != nil {
			return err
		}
	}
	return removeStaleLayoutFiles(filepath.Join(dir, profilesDirName), want)
}

// safeName keeps an ID usable as a file name on every platform. IDs are
// UUIDs today; anything else is reduced to a conservative character set.
func safeName(id string) string {
	if id == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func writeFileIfChanged(path string, data []byte) error {
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// removeStaleLayoutFiles deletes the JSON files under profiles/ that the
// current dataset no longer produces, then the folders left empty.
func removeStaleLayoutFiles(root string, keep map[string][]byte) error {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return fs.SkipDir
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		if _, ok := keep[path]; ok {
			return nil
		}
		return os.Remove(path)
	})
	if err != nil {
		return err
	}
	// Deepest folders first, so a profile folder empties after its tasks/.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if d == root {
			continue
		}
		if entries, err := os.ReadDir(d); err == nil && len(entries) == 0 {
			if err := os.Remove(d); err != nil {
				return err
			}
		}
	}
	return nil
}

// migrationCommitMessage marks the commit that converts a repository from
// the single todo.json to the folder layout.
const migrationCommitMessage = "sync: migration vers un fichier par tâche (format 3)"
