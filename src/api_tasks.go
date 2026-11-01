package src

func (a *App) ListTasks() ([]Task, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.store.listTasks()
}

func (a *App) CreateTask(input TaskInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.createTask(input)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) UpdateTask(id string, input TaskInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.updateTask(id, input)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) SetTaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.setDone(id, done)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

func (a *App) DeleteTask(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	if err := a.store.deleteTask(id); err != nil {
		return err
	}
	a.requestSync()
	return nil
}

func (a *App) SetTaskStarted(id string, started bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.setStarted(id, started)
	if err == nil {
		a.requestSync()
	}
	return task, err
}
