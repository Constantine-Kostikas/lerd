//go:build linux

package cli

// migrateWorkersOnModeChangeStreaming is the streaming variant; on Linux
// we still emit the bracketing events so the dashboard's progress UI
// completes, then return immediately.
func migrateWorkersOnModeChangeStreaming(_ /* fromMode */, _ /* toMode */ string, emit func(WorkerModePhaseEvent)) error {
	if emit != nil {
		emit(WorkerModePhaseEvent{Phase: "done"})
	}
	return nil
}
