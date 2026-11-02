package src

func (a *App) ListProfiles() ([]Profile, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.store.listProfiles()
}

func (a *App) CreateProfile(name string) (Profile, error) {
	if err := a.ready(); err != nil {
		return Profile{}, err
	}
	profile, err := a.store.createProfile(name)
	if err == nil {
		a.requestSync()
	}
	return profile, err
}

func (a *App) UpdateProfile(id, name, color string) (Profile, error) {
	if err := a.ready(); err != nil {
		return Profile{}, err
	}
	profile, err := a.store.updateProfile(id, name, color)
	if err == nil {
		a.requestSync()
	}
	return profile, err
}

func (a *App) DeleteProfile(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	if err := a.store.deleteProfile(id); err != nil {
		return err
	}
	a.requestSync()
	return nil
}
