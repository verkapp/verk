package src

func (a *App) AddSubtask(taskID, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.addSubtask(taskID, title)
}

func (a *App) AddSubtasks(taskID string, titles []string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.addSubtasks(taskID, titles)
}

func (a *App) UpdateSubtask(id, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.updateSubtask(id, title)
}

func (a *App) SetSubtaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.setSubtaskDone(id, done)
}

func (a *App) DeleteSubtask(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.deleteSubtask(id)
}
