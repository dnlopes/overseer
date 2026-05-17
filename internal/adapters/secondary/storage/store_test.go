package storage_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/dnlopes/overseer/internal/adapters/secondary/storage"
	"github.com/dnlopes/overseer/internal/core/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.New(filepath.Join(t.TempDir(), "sessions.json"), discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return store
}

func makeSession(t *testing.T, name, project string) domain.Session {
	t.Helper()
	s, err := domain.NewSession(name, project)
	if err != nil {
		t.Fatalf("domain.NewSession(%q, %q) error = %v", name, project, err)
	}
	return s
}

func TestNew_EmptyStoreHasNoSessions(t *testing.T) {
	store := newStore(t)
	sessions, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("List() len = %d, want 0", len(sessions))
	}
}

func TestSave_StoresSession(t *testing.T) {
	store := newStore(t)
	sess := makeSession(t, "alpha", "project-x")

	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestGet_ReturnsStoredSession(t *testing.T) {
	store := newStore(t)
	sess := makeSession(t, "alpha", "project-x")
	_ = store.Save(context.Background(), sess)

	got, err := store.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get() ID = %v, want %v", got.ID, sess.ID)
	}
	if got.Name != sess.Name {
		t.Errorf("Get() Name = %q, want %q", got.Name, sess.Name)
	}
	if got.ProjectName != sess.ProjectName {
		t.Errorf("Get() ProjectName = %q, want %q", got.ProjectName, sess.ProjectName)
	}
}

func TestGet_UnknownIDReturnsErrNotFound(t *testing.T) {
	store := newStore(t)

	_, err := store.Get(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Get() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestList_ReturnsAllSavedSessions(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	s1 := makeSession(t, "s1", "proj")
	s2 := makeSession(t, "s2", "proj")
	_ = store.Save(ctx, s1)
	_ = store.Save(ctx, s2)

	sessions, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("List() len = %d, want 2", len(sessions))
	}
}

func TestDelete_RemovesSession(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	sess := makeSession(t, "to-delete", "proj")
	_ = store.Save(ctx, sess)

	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Get(ctx, sess.ID)
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Get() after Delete() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestDelete_UnknownIDReturnsErrNotFound(t *testing.T) {
	store := newStore(t)

	err := store.Delete(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Delete() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestSave_OverwritesExistingEntry(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	sess := makeSession(t, "original", "proj")
	_ = store.Save(ctx, sess)

	_ = sess.Rename("renamed")
	_ = store.Save(ctx, sess)

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != "renamed" {
		t.Errorf("Get() Name = %q, want %q", got.Name, "renamed")
	}

	all, _ := store.List(ctx)
	if len(all) != 1 {
		t.Errorf("List() len = %d after overwrite, want 1", len(all))
	}
}

func TestPersistence_SurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	logger := discardLogger()
	ctx := context.Background()

	sess := makeSession(t, "persistent", "proj")

	store1, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := store1.Save(ctx, sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() (reload) error = %v", err)
	}
	got, err := store2.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.Name != sess.Name {
		t.Errorf("Get() Name = %q after reload, want %q", got.Name, sess.Name)
	}
}

func TestDelete_PersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	logger := discardLogger()
	ctx := context.Background()

	sess := makeSession(t, "ephemeral", "proj")

	store1, _ := storage.New(path, logger)
	_ = store1.Save(ctx, sess)
	_ = store1.Delete(ctx, sess.ID)

	store2, _ := storage.New(path, logger)
	all, err := store2.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 0 {
		t.Errorf("List() len = %d after reload post-delete, want 0", len(all))
	}
}
