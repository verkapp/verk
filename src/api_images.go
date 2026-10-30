package src

import wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

// AddImage opens the native picker, copies the chosen image into the
// application data directory and returns the refreshed task. Cancelling is not
// an error: the task is simply returned unchanged.
func (a *App) AddImage(taskID string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	if _, err := a.store.getTask(taskID); err != nil {
		return Task{}, err
	}
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choisir une image",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp"},
		},
	})
	if err != nil {
		return Task{}, err
	}
	if path == "" {
		return a.store.getTask(taskID)
	}
	return a.store.importImage(taskID, path)
}

func (a *App) DeleteImage(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.deleteImage(id)
}
