package task_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/task"
)

// setupStore creates an in-memory SQLite DB and returns a task store.
func setupStore(t *testing.T) task.PersistentTaskStore {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// Create table schema (copied from SQLTaskStore or schema definition)
	// We need to ensure the schema matches what SQLTaskStore expects.
	// Since we don't have migration machinery here, we manually create it.
	// The schema is:
	query := `
	CREATE TABLE IF NOT EXISTS a2a_tasks (
		id TEXT NOT NULL,
		app_name TEXT NOT NULL DEFAULT '',
		context_id TEXT NOT NULL,
		status_json TEXT NOT NULL,
		history_json TEXT NOT NULL,
		artifacts_json TEXT NOT NULL,
		metadata_json TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (app_name, id)
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_app_name ON a2a_tasks(app_name);
	CREATE INDEX IF NOT EXISTS idx_tasks_context_id ON a2a_tasks(context_id);
	`
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	store, err := task.NewSQLTaskStore(db, "sqlite")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store.(task.PersistentTaskStore)
}

func TestPersistentService_Create(t *testing.T) {
	store := setupStore(t)
	svc := task.NewPersistentService(store)
	ctx := app.WithAppID(context.Background(), "app-1")

	tsk, err := svc.Create(ctx, "ctx-1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if tsk.ID == "" {
		t.Error("Task ID empty")
	}
	if tsk.AppName != "app-1" {
		t.Errorf("AppName = %q, want app-1", tsk.AppName)
	}

	// Verify persistence
	loaded, err := svc.Get(ctx, tsk.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if loaded.ID != tsk.ID {
		t.Error("Loaded ID mismatch")
	}
}

func TestPersistentService_List(t *testing.T) {
	store := setupStore(t)
	svc := task.NewPersistentService(store)
	ctx1 := app.WithAppID(context.Background(), "app-1")

	_, _ = svc.Create(ctx1, "session-1")
	_, _ = svc.Create(ctx1, "session-1")
	_, _ = svc.Create(ctx1, "session-2")

	list, err := svc.List(ctx1, "session-1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List returned %d tasks, want 2", len(list))
	}
}

func TestPersistentService_Isolation(t *testing.T) {
	store := setupStore(t)
	svc := task.NewPersistentService(store)

	ctx1 := app.WithAppID(context.Background(), "app-1")
	ctx2 := app.WithAppID(context.Background(), "app-2")

	t1, err := svc.Create(ctx1, "session-common")
	if err != nil {
		t.Fatalf("Create t1 failed: %v", err)
	}
	t2, err := svc.Create(ctx2, "session-common")
	if err != nil {
		t.Fatalf("Create t2 failed: %v", err)
	}

	// App 1 should only see t1
	list1, _ := svc.List(ctx1, "session-common")
	if len(list1) != 1 || list1[0].ID != t1.ID {
		t.Errorf("App 1 isolation failed")
	}

	// App 2 should only see t2
	list2, _ := svc.List(ctx2, "session-common")
	if len(list2) != 1 || list2[0].ID != t2.ID {
		t.Errorf("App 2 isolation failed")
	}

	// App 1 cannot get t2
	_, err = svc.Get(ctx1, t2.ID)
	if err == nil {
		t.Error("App 1 should not match t2")
	}
}
