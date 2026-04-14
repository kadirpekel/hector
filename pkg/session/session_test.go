package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/session"
)

// =============================================================================
// InMemoryService Create Tests
// =============================================================================

func TestInMemoryService_Create(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, err := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if resp.Session == nil {
		t.Fatal("Session should not be nil")
	}
	if resp.Session.ID() != "session-1" {
		t.Errorf("ID() = %q, want session-1", resp.Session.ID())
	}
	if resp.Session.AppName() != "test-app" {
		t.Errorf("AppName() = %q, want test-app", resp.Session.AppName())
	}
	if resp.Session.UserID() != "user-1" {
		t.Errorf("UserID() = %q, want user-1", resp.Session.UserID())
	}
}

func TestInMemoryService_Create_GeneratesSessionID(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, err := svc.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		// SessionID empty - should be auto-generated
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if resp.Session.ID() == "" {
		t.Error("Session ID should be auto-generated")
	}
}

func TestInMemoryService_Create_WithInitialState(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, err := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
		State: map[string]any{
			"key1": "value1",
			"key2": 42,
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify initial state was set
	val, err := resp.Session.State().Get("key1")
	if err != nil {
		t.Errorf("State().Get(key1) error = %v", err)
	}
	if val != "value1" {
		t.Errorf("key1 = %v, want value1", val)
	}

	val2, _ := resp.Session.State().Get("key2")
	if val2 != 42 {
		t.Errorf("key2 = %v, want 42", val2)
	}
}

// =============================================================================
// InMemoryService Get Tests
// =============================================================================

func TestInMemoryService_Get(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	// Create a session
	_, err := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Retrieve it
	resp, err := svc.Get(ctx, &session.GetRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if resp.Session.ID() != "session-1" {
		t.Errorf("ID() = %q, want session-1", resp.Session.ID())
	}
}

func TestInMemoryService_Get_NotFound(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	_, err := svc.Get(ctx, &session.GetRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "nonexistent",
	})

	if err != session.ErrSessionNotFound {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

// =============================================================================
// InMemoryService List Tests
// =============================================================================

func TestInMemoryService_List(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	// Create multiple sessions
	for i := 1; i <= 3; i++ {
		_, err := svc.Create(ctx, &session.CreateRequest{
			AppName:   "test-app",
			UserID:    "user-1",
			SessionID: "session-" + string(rune('0'+i)),
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Create session for different user
	_, _ = svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-2",
		SessionID: "session-other",
	})

	// List sessions for user-1
	resp, err := svc.List(ctx, &session.ListRequest{
		AppName: "test-app",
		UserID:  "user-1",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(resp.Sessions) != 3 {
		t.Errorf("Expected 3 sessions for user-1, got %d", len(resp.Sessions))
	}
}

func TestInMemoryService_List_Empty(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, err := svc.List(ctx, &session.ListRequest{
		AppName: "test-app",
		UserID:  "user-1",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(resp.Sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(resp.Sessions))
	}
}

// =============================================================================
// InMemoryService Delete Tests
// =============================================================================

func TestInMemoryService_Delete(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	// Create and then delete
	_, _ = svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	err := svc.Delete(ctx, &session.DeleteRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's deleted
	_, err = svc.Get(ctx, &session.GetRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})
	if err != session.ErrSessionNotFound {
		t.Error("Session should be deleted")
	}
}

func TestInMemoryService_Delete_Nonexistent(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	// Deleting nonexistent session should not error
	err := svc.Delete(ctx, &session.DeleteRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "nonexistent",
	})
	if err != nil {
		t.Errorf("Delete() should not error for nonexistent session: %v", err)
	}
}

// =============================================================================
// Session State Tests
// =============================================================================

func TestSession_State_GetSet(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	state := resp.Session.State()

	// Set a value
	if err := state.Set("mykey", "myvalue"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the value
	val, err := state.Get("mykey")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if val != "myvalue" {
		t.Errorf("Get(mykey) = %v, want myvalue", val)
	}
}

func TestSession_State_GetNotExist(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	_, err := resp.Session.State().Get("nonexistent")
	if err != session.ErrStateKeyNotExist {
		t.Errorf("Get(nonexistent) error = %v, want ErrStateKeyNotExist", err)
	}
}

func TestSession_State_Delete(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	state := resp.Session.State()
	_ = state.Set("key", "value")
	_ = state.Delete("key")

	_, err := state.Get("key")
	if err != session.ErrStateKeyNotExist {
		t.Error("Key should be deleted")
	}
}

// =============================================================================
// Session Events Tests
// =============================================================================

func TestInMemoryService_AppendEvent(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	createResp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	event := &agent.Event{
		ID:        "event-1",
		Timestamp: time.Now(),
	}

	err := svc.AppendEvent(ctx, createResp.Session, event)
	if err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	// Verify event was added
	events := createResp.Session.Events()
	if events.Len() != 1 {
		t.Errorf("Events.Len() = %d, want 1", events.Len())
	}
	if events.At(0).ID != "event-1" {
		t.Errorf("Events.At(0).ID = %q, want event-1", events.At(0).ID)
	}
}

func TestInMemoryService_AppendEvent_UpdatesTimestamp(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	createResp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	timeBefore := createResp.Session.LastUpdateTime()
	time.Sleep(10 * time.Millisecond)

	if err := svc.AppendEvent(ctx, createResp.Session, &agent.Event{ID: "event-1"}); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	timeAfter := createResp.Session.LastUpdateTime()
	if !timeAfter.After(timeBefore) {
		t.Error("LastUpdateTime should be updated after AppendEvent")
	}
}

func TestSession_Events_At_OutOfBounds(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	events := resp.Session.Events()
	if events.At(-1) != nil {
		t.Error("At(-1) should return nil")
	}
	if events.At(100) != nil {
		t.Error("At(100) should return nil for empty events")
	}
}

// =============================================================================
// Key Prefix Constants Tests
// =============================================================================

func TestKeyPrefixConstants(t *testing.T) {
	// Verify prefix constants exist and have expected values
	if session.KeyPrefixApp != "app:" {
		t.Errorf("KeyPrefixApp = %q, want app:", session.KeyPrefixApp)
	}
	if session.KeyPrefixUser != "user:" {
		t.Errorf("KeyPrefixUser = %q, want user:", session.KeyPrefixUser)
	}
	if session.KeyPrefixTemp != "temp:" {
		t.Errorf("KeyPrefixTemp = %q, want temp:", session.KeyPrefixTemp)
	}
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestInMemoryService_InterfaceCompliance(t *testing.T) {
	var _ session.Service = session.InMemoryService()
}

func TestInMemorySession_InterfaceCompliance(t *testing.T) {
	svc := session.InMemoryService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
	})

	var _ session.Session = resp.Session
}
