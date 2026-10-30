package src

func (a *App) ClaudeStatus() ClaudeStatus {
	if err := a.ready(); err != nil {
		return ClaudeStatus{Error: err.Error()}
	}
	return a.claudeStatus()
}

func (a *App) BreakDownTask(taskID string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.breakDownTask(taskID)
}

func (a *App) CoachNudge() (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.coachNudge()
}
