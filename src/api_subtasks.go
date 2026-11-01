package src

func (a *App) AddSubtask(taskID, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.addSubtask(taskID, title)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) AddSubtasks(taskID string, titles []string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.addSubtasks(taskID, titles)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) UpdateSubtask(id, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.updateSubtask(id, title)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) SetSubtaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.setSubtaskDone(id, done)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) DeleteSubtask(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.deleteSubtask(id)
	if err == nil {
		a.requestSync()
	}
	return task, err
}
