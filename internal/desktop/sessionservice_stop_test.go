package desktop

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

type fakeManagedSessionStopper struct {
	mu              sync.Mutex
	sessions        []session.Info
	terminateErrors map[string]error
	terminateCalls  []string
	stopCalls       int
	listCalls       int
	listErrors      map[int]error
	stopErr         error
	onTerminate     func(id string, sessions []session.Info) []session.Info
}

func (f *fakeManagedSessionStopper) List() ([]session.Info, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	if err := f.listErrors[f.listCalls]; err != nil {
		return nil, err
	}
	return append([]session.Info(nil), f.sessions...), nil
}

func (f *fakeManagedSessionStopper) Terminate(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.terminateCalls = append(f.terminateCalls, id)
	if err := f.terminateErrors[id]; err != nil {
		return err
	}
	for i := range f.sessions {
		if f.sessions[i].ID == id && f.sessions[i].Status == "running" {
			f.sessions[i].Status = "exited"
		}
	}
	if f.onTerminate != nil {
		f.sessions = f.onTerminate(id, append([]session.Info(nil), f.sessions...))
	}
	return nil
}

func (f *fakeManagedSessionStopper) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	return f.stopErr
}

func TestRunStopAllAfterConfirmationCancelDoesNothing(t *testing.T) {
	called := false
	err := runStopAllAfterConfirmation(false, []string{"a", "b"}, func([]string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("cancel invoked the session termination operation")
	}
}

func TestStopManagedSessionsRequiresConfirmedCurrentSet(t *testing.T) {
	backend := &fakeManagedSessionStopper{sessions: []session.Info{
		{ID: "a", Status: "running"}, {ID: "b", Status: "running"},
	}}
	err := stopManagedSessionsAndDaemon(backend, []string{"a"})
	var changed *managedSessionStateChangedError
	if !errors.As(err, &changed) || !reflect.DeepEqual(changed.currentIDs, []string{"a", "b"}) {
		t.Fatalf("expected current managed IDs in state-change error, got %v", err)
	}
	if len(backend.terminateCalls) != 0 || backend.stopCalls != 0 {
		t.Fatalf("changed count caused actions: terminate=%v stop=%d", backend.terminateCalls, backend.stopCalls)
	}
}

func TestStopManagedSessionsRejectsSameCountWithDifferentSessionIdentity(t *testing.T) {
	backend := &fakeManagedSessionStopper{sessions: []session.Info{
		{ID: "replacement", Status: "running"},
	}}
	err := stopManagedSessionsAndDaemon(backend, []string{"confirmed-before-dialog"})
	var changed *managedSessionStateChangedError
	if !errors.As(err, &changed) || !reflect.DeepEqual(changed.currentIDs, []string{"replacement"}) {
		t.Fatalf("expected identity revalidation despite matching count, got %v", err)
	}
	if len(backend.terminateCalls) != 0 || backend.stopCalls != 0 {
		t.Fatalf("replacement session was acted on without confirmation: terminate=%v stop=%d", backend.terminateCalls, backend.stopCalls)
	}
}

func TestStopManagedSessionsTerminatesOnlyRunningDaemonSessions(t *testing.T) {
	backend := &fakeManagedSessionStopper{sessions: []session.Info{
		{ID: "managed-a", Status: "running"},
		{ID: "already-exited", Status: "exited"},
		{ID: "managed-b", Status: "running"},
	}}
	if err := stopManagedSessionsAndDaemon(backend, []string{"managed-b", "managed-a"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(backend.terminateCalls, []string{"managed-a", "managed-b"}) &&
		!reflect.DeepEqual(backend.terminateCalls, []string{"managed-b", "managed-a"}) {
		t.Fatalf("unexpected terminate targets: %v", backend.terminateCalls)
	}
	if backend.stopCalls != 1 {
		t.Fatalf("Stop calls=%d, want 1", backend.stopCalls)
	}
}

func TestStopManagedSessionsPartialTerminationFailureKeepsDaemonRunning(t *testing.T) {
	backend := &fakeManagedSessionStopper{
		sessions:        []session.Info{{ID: "a", Status: "running"}, {ID: "b", Status: "running"}},
		terminateErrors: map[string]error{"b": errors.New("termination denied")},
	}
	err := stopManagedSessionsAndDaemon(backend, []string{"a", "b"})
	if err == nil || !strings.Contains(err.Error(), "termination denied") {
		t.Fatalf("expected termination error, got %v", err)
	}
	if backend.stopCalls != 0 {
		t.Fatalf("daemon stopped after partial termination failure: %d", backend.stopCalls)
	}
}

func TestStopManagedSessionsDoesNotTerminateConcurrentUnconfirmedSession(t *testing.T) {
	backend := &fakeManagedSessionStopper{
		sessions: []session.Info{{ID: "confirmed", Status: "running"}},
		onTerminate: func(id string, items []session.Info) []session.Info {
			return append(items, session.Info{ID: "concurrent", Status: "running"})
		},
	}
	err := stopManagedSessionsAndDaemon(backend, []string{"confirmed"})
	var changed *managedSessionStateChangedError
	if !errors.As(err, &changed) || !reflect.DeepEqual(changed.currentIDs, []string{"concurrent"}) {
		t.Fatalf("expected re-confirmation after concurrent session creation, got %v", err)
	}
	if !reflect.DeepEqual(backend.terminateCalls, []string{"confirmed"}) {
		t.Fatalf("unconfirmed session was targeted: %v", backend.terminateCalls)
	}
	if backend.stopCalls != 0 {
		t.Fatalf("daemon stopped with an unconfirmed running session: %d", backend.stopCalls)
	}
}

func TestStopManagedSessionsAbortsOnDisconnectAndDaemonStopError(t *testing.T) {
	t.Run("list disconnect", func(t *testing.T) {
		backend := &fakeManagedSessionStopper{
			sessions:   []session.Info{{ID: "a", Status: "running"}},
			listErrors: map[int]error{2: errors.New("disconnected")},
		}
		if err := stopManagedSessionsAndDaemon(backend, []string{"a"}); err == nil {
			t.Fatal("disconnected backend returned nil")
		}
		if backend.stopCalls != 0 {
			t.Fatalf("daemon stop called after disconnect: %d", backend.stopCalls)
		}
	})
	t.Run("stop refused", func(t *testing.T) {
		backend := &fakeManagedSessionStopper{
			sessions: []session.Info{{ID: "a", Status: "running"}},
			stopErr:  fmt.Errorf("stop refused by daemon guard"),
		}
		if err := stopManagedSessionsAndDaemon(backend, []string{"a"}); err == nil {
			t.Fatal("daemon stop guard error was swallowed")
		}
		if backend.stopCalls != 1 {
			t.Fatalf("Stop calls=%d, want 1", backend.stopCalls)
		}
	})
}

func TestBulkStopGuardRejectsConcurrentActions(t *testing.T) {
	service := &SessionService{}
	if !service.tryBeginBulkStop() {
		t.Fatal("first stop-all action was rejected")
	}
	if service.tryBeginBulkStop() {
		service.finishBulkStop()
		t.Fatal("concurrent stop-all action was accepted")
	}
	service.finishBulkStop()
	if !service.tryBeginBulkStop() {
		t.Fatal("stop-all guard did not reset after completion")
	}
	service.finishBulkStop()
}
