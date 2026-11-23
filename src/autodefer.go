package src

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Auto-defer: every profile can set a time of day ("18:00") after which its
// unfinished tasks for the day are pushed to the next working day, exactly
// like the "Reprendre demain / lundi" button.

const autoDeferTick = 30 * time.Second

// autoDeferKey stores, per machine, when a profile was last processed. It is
// kept in the local settings table (not synced) so each machine runs once per
// cutoff on its own clock.
func autoDeferKey(profileID string) string {
	return "auto_defer_last_" + profileID
}

// parseClock accepts "H:MM" or "HH:MM" and returns the normalised "HH:MM".
func parseClock(value string) (hour, minute int, normalised string, err error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, "", fmt.Errorf("heure invalide : %q", value)
	}
	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 || len(parts[1]) != 2 {
		return 0, 0, "", fmt.Errorf("heure invalide : %q", value)
	}
	return hour, minute, fmt.Sprintf("%02d:%02d", hour, minute), nil
}

// lastCutoff is the most recent occurrence of hour:minute at or before now.
func lastCutoff(now time.Time, hour, minute int) time.Time {
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if cutoff.After(now) {
		cutoff = cutoff.AddDate(0, 0, -1)
	}
	return cutoff
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// resumeDay mirrors resumeDay() in frontend/src/app/dates.ts: Monday when the
// weekend is ahead (Friday, Saturday), the next day otherwise.
func resumeDay(from time.Time) time.Time {
	ahead := 1
	switch from.Weekday() {
	case time.Friday:
		ahead = 3
	case time.Saturday:
		ahead = 2
	}
	return startOfDay(from).AddDate(0, 0, ahead)
}

// hasSpecificTime mirrors the frontend: 23:59 means "some time that day".
func hasSpecificTime(t time.Time) bool {
	return !(t.Hour() == 23 && t.Minute() == 59)
}

// resumeDueAt keeps a specific time of day, otherwise ends the day at 23:59,
// like resumeDueAt() in the frontend.
func resumeDueAt(due time.Time, day time.Time) time.Time {
	if hasSpecificTime(due) {
		return time.Date(day.Year(), day.Month(), day.Day(), due.Hour(), due.Minute(), 0, 0, day.Location())
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 0, 0, day.Location())
}

// autoDeferMoves returns the new due date of every task the cutoff pushes:
// open tasks of the profile that are overdue, or due on the cutoff day
// either without a specific time or at a time not after the cutoff. Tasks
// planned later that evening, and tasks without a due date, stay put.
func autoDeferMoves(tasks []Task, profileID string, cutoff time.Time) map[string]int64 {
	moves := map[string]int64{}
	cutoffDay := startOfDay(cutoff)
	target := resumeDay(cutoff)
	for _, t := range tasks {
		if t.Done || t.ProfileID != profileID || t.DueAt == nil {
			continue
		}
		due := time.UnixMilli(*t.DueAt).In(cutoff.Location())
		dueDay := startOfDay(due)
		switch {
		case dueDay.Before(cutoffDay):
		case dueDay.Equal(cutoffDay) && (!hasSpecificTime(due) || !due.After(cutoff)):
		default:
			continue
		}
		moves[t.ID] = resumeDueAt(due, target).UnixMilli()
	}
	return moves
}

// runAutoDefer processes every profile whose cutoff has passed since its last
// run. It returns how many tasks moved and the day they moved to.
func (a *App) runAutoDefer(now time.Time) (int, time.Time, error) {
	profiles, err := a.store.listProfiles()
	if err != nil {
		return 0, time.Time{}, err
	}
	var tasks []Task
	moved := 0
	var target time.Time
	for _, p := range profiles {
		if p.AutoDeferAt == "" {
			continue
		}
		hour, minute, _, err := parseClock(p.AutoDeferAt)
		if err != nil {
			continue
		}
		key := autoDeferKey(p.ID)
		last, ok, err := a.store.settingInt(key)
		if err != nil {
			return moved, target, err
		}
		stamp := map[string]string{key: strconv.FormatInt(now.UnixMilli(), 10)}
		if !ok {
			// First run on this machine: only future cutoffs apply.
			if err := a.store.saveSettingValues(stamp); err != nil {
				return moved, target, err
			}
			continue
		}
		cutoff := lastCutoff(now, hour, minute)
		if last >= cutoff.UnixMilli() {
			continue
		}
		if tasks == nil {
			if tasks, err = a.store.listTasks(); err != nil {
				return moved, target, err
			}
		}
		moves := autoDeferMoves(tasks, p.ID, cutoff)
		if len(moves) > 0 {
			if err := a.store.deferTasks(moves, now.UnixMilli()); err != nil {
				return moved, target, err
			}
			moved += len(moves)
			target = resumeDay(cutoff)
		}
		if err := a.store.saveSettingValues(stamp); err != nil {
			return moved, target, err
		}
	}
	return moved, target, nil
}

// autoDeferLoop checks the cutoffs at startup (catching up on a cutoff
// missed while the app was closed) and then every autoDeferTick.
func (a *App) autoDeferLoop() {
	ticker := time.NewTicker(autoDeferTick)
	defer ticker.Stop()
	for {
		a.autoDeferOnce()
		select {
		case <-a.autoDeferStop:
			return
		case <-ticker.C:
		}
	}
}

func (a *App) autoDeferOnce() {
	now := time.Now()
	moved, target, err := a.runAutoDefer(now)
	if err != nil || moved == 0 {
		return
	}
	a.requestSync()
	a.emit("tasks:deferred", map[string]any{"count": moved, "label": dayLabel(target, now)})
}

var weekdaysFR = [...]string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}

// dayLabel names the target day for the toast: "aujourd'hui" when catching
// up in the morning, "demain", or the weekday ("lundi").
func dayLabel(day, now time.Time) string {
	switch startOfDay(day).Sub(startOfDay(now)).Hours() / 24 {
	case 0:
		return "aujourd’hui"
	case 1:
		return "demain"
	default:
		return weekdaysFR[day.Weekday()]
	}
}
