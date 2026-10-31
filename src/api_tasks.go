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
	return a.store.createTask(input)
}

func (a *App) UpdateTask(id string, input TaskInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.updateTask(id, input)
}

func (a *App) SetTaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.setDone(id, done)
}

func (a *App) DeleteTask(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.store.deleteTask(id)
}

func (a *App) SetTaskStarted(id string, started bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.setStarted(id, started)
}
