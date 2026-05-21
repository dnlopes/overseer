package storage_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
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
	store, err := storage.New(filepath.Join(t.TempDir(), "data.json"), discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return store
}

func makeSession(t *testing.T, name string, projectID uuid.UUID) domain.Session {
	t.Helper()
	s, err := domain.NewSession(name, projectID)
	if err != nil {
		t.Fatalf("domain.NewSession(%q, %v) error = %v", name, projectID, err)
	}
	return s
}

func makeProject(t *testing.T, path, name string) domain.Project {
	t.Helper()
	p, err := domain.NewProject(path, name)
	if err != nil {
		t.Fatalf("domain.NewProject(%q, %q) error = %v", path, name, err)
	}
	return p
}

// --- SessionStore ---

func TestSessionStore_EmptyStoreHasNoSessions(t *testing.T) {
	store := newStore(t).Sessions()
	sessions, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("List() len = %d, want 0", len(sessions))
	}
}

func TestSessionStore_SaveStoresSession(t *testing.T) {
	store := newStore(t).Sessions()
	sess := makeSession(t, "alpha", uuid.New())

	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestSessionStore_GetReturnsStoredSession(t *testing.T) {
	projectID := uuid.New()
	store := newStore(t).Sessions()
	sess := makeSession(t, "alpha", projectID)
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
	if got.ProjectID != projectID {
		t.Errorf("Get() ProjectID = %v, want %v", got.ProjectID, projectID)
	}
}

func TestSessionStore_GetUnknownIDReturnsErrNotFound(t *testing.T) {
	store := newStore(t).Sessions()

	_, err := store.Get(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Get() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestSessionStore_ListReturnsAllSavedSessions(t *testing.T) {
	store := newStore(t).Sessions()
	ctx := context.Background()
	projectID := uuid.New()

	s1 := makeSession(t, "s1", projectID)
	s2 := makeSession(t, "s2", projectID)
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

func TestSessionStore_DeleteRemovesSession(t *testing.T) {
	store := newStore(t).Sessions()
	ctx := context.Background()
	sess := makeSession(t, "to-delete", uuid.New())
	_ = store.Save(ctx, sess)

	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Get(ctx, sess.ID)
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Get() after Delete() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestSessionStore_DeleteUnknownIDReturnsErrNotFound(t *testing.T) {
	store := newStore(t).Sessions()

	err := store.Delete(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Delete() error = %v, want domain.ErrSessionNotFound", err)
	}
}

func TestSessionStore_SaveOverwritesExistingEntry(t *testing.T) {
	store := newStore(t).Sessions()
	ctx := context.Background()
	sess := makeSession(t, "original", uuid.New())
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

func TestSessionStore_PersistenceSurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()
	projectID := uuid.New()

	sess := makeSession(t, "persistent", projectID)

	store1, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := store1.Sessions().Save(ctx, sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() (reload) error = %v", err)
	}
	got, err := store2.Sessions().Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.Name != sess.Name {
		t.Errorf("Get() Name = %q after reload, want %q", got.Name, sess.Name)
	}
	if got.ProjectID != projectID {
		t.Errorf("Get() ProjectID = %v after reload, want %v", got.ProjectID, projectID)
	}
}

func TestSessionStore_WorktreeFieldsSurviveReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()
	projectID := uuid.New()

	sess := makeSession(t, "with-worktree", projectID)
	if err := sess.AssignWorktree("/abs/worktrees/alpha", "main", "overseer/alpha"); err != nil {
		t.Fatalf("AssignWorktree() error = %v", err)
	}

	store1, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := store1.Sessions().Save(ctx, sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() (reload) error = %v", err)
	}
	got, err := store2.Sessions().Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.WorktreePath != "/abs/worktrees/alpha" {
		t.Errorf("Get() WorktreePath = %q, want %q", got.WorktreePath, "/abs/worktrees/alpha")
	}
	if got.BaseBranch != "main" {
		t.Errorf("Get() BaseBranch = %q, want %q", got.BaseBranch, "main")
	}
	if got.FeatureBranch != "overseer/alpha" {
		t.Errorf("Get() FeatureBranch = %q, want %q", got.FeatureBranch, "overseer/alpha")
	}
	if !got.HasWorktree() {
		t.Errorf("Get() HasWorktree() = false, want true")
	}
}

func TestSessionStore_AgentCommandSurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()

	sess := makeSession(t, "with-agent", uuid.New())
	if err := sess.AssignAgentCommand("opencode --config /tmp/cfg.json"); err != nil {
		t.Fatalf("AssignAgentCommand() error = %v", err)
	}

	store1, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := store1.Sessions().Save(ctx, sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, err := storage.New(path, logger)
	if err != nil {
		t.Fatalf("New() (reload) error = %v", err)
	}
	got, err := store2.Sessions().Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.AgentCommand != "opencode --config /tmp/cfg.json" {
		t.Errorf("Get() AgentCommand = %q, want %q", got.AgentCommand, "opencode --config /tmp/cfg.json")
	}
}

func TestSessionStore_LegacySessionsHaveEmptyAgentCommandAfterReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()

	sess := makeSession(t, "legacy", uuid.New())

	store1, _ := storage.New(path, logger)
	if err := store1.Sessions().Save(ctx, sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, _ := storage.New(path, logger)
	got, err := store2.Sessions().Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.AgentCommand != "" {
		t.Errorf("Get() AgentCommand = %q, want empty (legacy session)", got.AgentCommand)
	}
}

func TestSessionStore_DeletePersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()

	sess := makeSession(t, "ephemeral", uuid.New())

	store1, _ := storage.New(path, logger)
	_ = store1.Sessions().Save(ctx, sess)
	_ = store1.Sessions().Delete(ctx, sess.ID)

	store2, _ := storage.New(path, logger)
	all, _ := store2.Sessions().List(ctx)
	if len(all) != 0 {
		t.Errorf("List() len = %d after delete+reload, want 0", len(all))
	}
}

// --- ProjectStore ---

func TestProjectStore_EmptyStoreHasNoProjects(t *testing.T) {
	store := newStore(t).Projects()
	projects, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("List() len = %d, want 0", len(projects))
	}
}

func TestProjectStore_SaveAndGet(t *testing.T) {
	store := newStore(t).Projects()
	project := makeProject(t, "/repo/overseer", "Overseer")

	if err := store.Save(context.Background(), project); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Path != "/repo/overseer" {
		t.Errorf("Get() Path = %q, want %q", got.Path, "/repo/overseer")
	}
	if got.Name != "Overseer" {
		t.Errorf("Get() Name = %q, want %q", got.Name, "Overseer")
	}
}

func TestProjectStore_GetByPathFindsExistingProject(t *testing.T) {
	store := newStore(t).Projects()
	project := makeProject(t, "/repo/widgets", "Widgets")
	_ = store.Save(context.Background(), project)

	got, err := store.GetByPath(context.Background(), "/repo/widgets")
	if err != nil {
		t.Fatalf("GetByPath() error = %v", err)
	}
	if got.ID != project.ID {
		t.Errorf("GetByPath() ID = %v, want %v", got.ID, project.ID)
	}
}

func TestProjectStore_GetByPathReturnsNotFoundForUnknownPath(t *testing.T) {
	store := newStore(t).Projects()

	_, err := store.GetByPath(context.Background(), "/nonexistent")
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("GetByPath() error = %v, want domain.ErrProjectNotFound", err)
	}
}

func TestProjectStore_GetUnknownIDReturnsErrNotFound(t *testing.T) {
	store := newStore(t).Projects()

	_, err := store.Get(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("Get() error = %v, want domain.ErrProjectNotFound", err)
	}
}

func TestProjectStore_DeleteRemovesProject(t *testing.T) {
	store := newStore(t).Projects()
	project := makeProject(t, "/repo/x", "X")
	_ = store.Save(context.Background(), project)

	if err := store.Delete(context.Background(), project.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err := store.Get(context.Background(), project.ID)
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("Get() after Delete() error = %v, want domain.ErrProjectNotFound", err)
	}
}

func TestProjectStore_PersistenceSurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()

	project := makeProject(t, "/repo/persist", "Persist")

	store1, _ := storage.New(path, logger)
	_ = store1.Projects().Save(ctx, project)

	store2, _ := storage.New(path, logger)
	got, err := store2.Projects().Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.Name != "Persist" {
		t.Errorf("Get() Name = %q after reload, want %q", got.Name, "Persist")
	}
}

// --- Cross-aggregate persistence ---

func TestStore_PersistsBothAggregatesInSameFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	logger := discardLogger()
	ctx := context.Background()

	project := makeProject(t, "/repo/overseer", "Overseer")
	sess := makeSession(t, "alpha", project.ID)

	store1, _ := storage.New(path, logger)
	_ = store1.Projects().Save(ctx, project)
	_ = store1.Sessions().Save(ctx, sess)

	store2, _ := storage.New(path, logger)
	gotProject, err := store2.Projects().Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get(project) after reload error = %v", err)
	}
	if gotProject.Name != "Overseer" {
		t.Errorf("Project name lost across reload: %q", gotProject.Name)
	}

	gotSession, err := store2.Sessions().Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get(session) after reload error = %v", err)
	}
	if gotSession.ProjectID != project.ID {
		t.Errorf("Session ProjectID = %v after reload, want %v", gotSession.ProjectID, project.ID)
	}
}

func TestStore_LegacyOpenBranchKindMigratesToCheckoutOnLoad(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data.json")
	legacy := `{
  "projects": [],
  "sessions": [
    {
      "ID": "00000000-0000-0000-0000-00000000abcd",
      "Name": "main",
      "ProjectID": "00000000-0000-0000-0000-00000000beef",
      "Kind": "open-branch",
      "Order": 1,
      "WorktreePath": "/tmp/wt/abcd",
      "BaseBranch": "main",
      "FeatureBranch": "overseer/check/abcd",
      "AgentCommand": "",
      "EditorCommand": "",
      "CreatedAt": "2025-01-01T00:00:00Z",
      "UpdatedAt": "2025-01-01T00:00:00Z"
    }
  ]
}`
	if err := os.WriteFile(dataPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("seed legacy file: %v", err)
	}

	store, err := storage.New(dataPath, discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sess, err := store.Sessions().Get(context.Background(), uuid.MustParse("00000000-0000-0000-0000-00000000abcd"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if sess.Kind != domain.SessionKindCheckout {
		t.Fatalf("legacy Kind not migrated: got %q, want %q", sess.Kind, domain.SessionKindCheckout)
	}
	if !sess.IsCheckout() {
		t.Fatal("legacy session should report IsCheckout() == true after migration")
	}
}
