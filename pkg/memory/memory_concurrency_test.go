package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// Mock implementations for testing

type mockSessionService struct {
	mu       sync.Mutex
	messages map[string][]*pb.Message
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		messages: make(map[string][]*pb.Message),
	}
}

func (m *mockSessionService) AppendMessage(sessionID string, message *pb.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], message)
	return nil
}

func (m *mockSessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], messages...)
	return nil
}

func (m *mockSessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msgs := m.messages[sessionID]
	if limit > 0 && len(msgs) > limit {
		return msgs[len(msgs)-limit:], nil
	}
	return msgs, nil
}

func (m *mockSessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {
	return m.GetMessages(sessionID, opts.Limit)
}

func (m *mockSessionService) GetMessageCount(sessionID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages[sessionID]), nil
}

func (m *mockSessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	return &reasoning.SessionMetadata{ID: sessionID}, nil
}

func (m *mockSessionService) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.messages, sessionID)
	return nil
}

func (m *mockSessionService) SessionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

type mockWorkingMemory struct{}

func (m *mockWorkingMemory) AddMessage(session *hectorcontext.ConversationHistory, message *pb.Message) error {
	return nil
}

func (m *mockWorkingMemory) CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	return nil, nil
}

func (m *mockWorkingMemory) GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	return session.GetAllMessages(), nil
}

func (m *mockWorkingMemory) SetStatusNotifier(notifier StatusNotifier) {}

func (m *mockWorkingMemory) Name() string { return "mock" }

func (m *mockWorkingMemory) LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error) {
	history, _ := hectorcontext.NewConversationHistory(sessionID)
	return history, nil
}

type mockLongTermMemory struct {
	mu      sync.Mutex
	stored  map[string][]*pb.Message
	storeCh chan bool // For synchronization in tests
}

func newMockLongTermMemory() *mockLongTermMemory {
	return &mockLongTermMemory{
		stored:  make(map[string][]*pb.Message),
		storeCh: make(chan bool, 100),
	}
}

func (m *mockLongTermMemory) Store(agentID string, sessionID string, messages []*pb.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := agentID + ":" + sessionID
	m.stored[key] = append(m.stored[key], messages...)

	// Signal that a store happened
	select {
	case m.storeCh <- true:
	default:
	}

	return nil
}

func (m *mockLongTermMemory) Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error) {
	return nil, nil
}

func (m *mockLongTermMemory) Clear(agentID string, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := agentID + ":" + sessionID
	delete(m.stored, key)
	return nil
}

func (m *mockLongTermMemory) Name() string {
	return "mock"
}

func (m *mockLongTermMemory) GetStoredCount(agentID string, sessionID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := agentID + ":" + sessionID
	return len(m.stored[key])
}

// Test concurrent AddBatchToHistory calls
func TestMemoryService_ConcurrentAddBatch(t *testing.T) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    5,
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	// Run 100 concurrent operations
	numGoroutines := 100
	messagesPerGoroutine := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			sessionID := "test-session"

			for j := 0; j < messagesPerGoroutine; j++ {
				messages := []*pb.Message{
					{
						Role: pb.Role_ROLE_USER,
						Content: []*pb.Part{
							{Part: &pb.Part_Text{Text: "test message"}},
						},
					},
				}

				if err := memoryService.AddBatchToHistory(sessionID, messages); err != nil {
					t.Errorf("AddBatchToHistory failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were saved
	count, err := sessionService.GetMessageCount("test-session")
	if err != nil {
		t.Fatalf("GetMessageCount failed: %v", err)
	}

	expected := numGoroutines * messagesPerGoroutine
	if count != expected {
		t.Errorf("Expected %d messages, got %d", expected, count)
	}

	t.Logf("✅ Concurrent test passed: %d messages from %d goroutines", count, numGoroutines)
}

// Test concurrent long-term memory batching
func TestMemoryService_ConcurrentLongTermBatching(t *testing.T) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    10, // Small batch size to trigger frequent flushes
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	// Run 50 concurrent operations
	numGoroutines := 50
	messagesPerGoroutine := 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			sessionID := "test-session"

			for j := 0; j < messagesPerGoroutine; j++ {
				messages := []*pb.Message{
					{
						Role: pb.Role_ROLE_USER,
						Content: []*pb.Part{
							{Part: &pb.Part_Text{Text: "test message"}},
						},
					},
				}

				_ = memoryService.AddBatchToHistory(sessionID, messages)

				// Small delay to increase likelihood of concurrent batch operations
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// Flush any remaining batches
	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify long-term memory received all messages
	stored := longTermMemory.GetStoredCount("test-agent", "test-session")
	expected := numGoroutines * messagesPerGoroutine

	if stored != expected {
		t.Errorf("Expected %d stored messages, got %d", expected, stored)
	}

	t.Logf("✅ Concurrent long-term batching passed: %d messages stored", stored)
}

// Test Shutdown with pending batches
func TestMemoryService_ShutdownWithPendingBatches(t *testing.T) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    100, // Large batch size to keep messages pending
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	// Add messages to multiple sessions
	sessions := []string{"session1", "session2", "session3"}
	messagesPerSession := 50 // Less than batch size, so they stay pending

	for _, sessionID := range sessions {
		for i := 0; i < messagesPerSession; i++ {
			messages := []*pb.Message{
				{
					Role: pb.Role_ROLE_USER,
					Content: []*pb.Part{
						{Part: &pb.Part_Text{Text: "test message"}},
					},
				},
			}
			_ = memoryService.AddBatchToHistory(sessionID, messages)
		}
	}

	// Before shutdown, nothing should be in long-term memory
	for _, sessionID := range sessions {
		stored := longTermMemory.GetStoredCount("test-agent", sessionID)
		if stored != 0 {
			t.Errorf("Expected 0 stored messages before shutdown for %s, got %d", sessionID, stored)
		}
	}

	// Shutdown should flush all pending batches
	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// After shutdown, all messages should be in long-term memory
	for _, sessionID := range sessions {
		stored := longTermMemory.GetStoredCount("test-agent", sessionID)
		if stored != messagesPerSession {
			t.Errorf("Expected %d stored messages after shutdown for %s, got %d",
				messagesPerSession, sessionID, stored)
		}
	}

	t.Logf("✅ Shutdown test passed: %d sessions, %d messages per session flushed",
		len(sessions), messagesPerSession)
}

// Test concurrent Shutdown calls (should be idempotent)
func TestMemoryService_ConcurrentShutdown(t *testing.T) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    100,
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	// Add some pending messages
	for i := 0; i < 50; i++ {
		messages := []*pb.Message{
			{
				Role: pb.Role_ROLE_USER,
				Content: []*pb.Part{
					{Part: &pb.Part_Text{Text: "test message"}},
				},
			},
		}
		_ = memoryService.AddBatchToHistory("test-session", messages)
	}

	// Call Shutdown from multiple goroutines
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if err := memoryService.Shutdown(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent shutdown returned error: %v", err)
		}
	}

	// All messages should be stored (exactly once, not duplicated)
	stored := longTermMemory.GetStoredCount("test-agent", "test-session")
	if stored != 50 {
		t.Errorf("Expected 50 stored messages, got %d (possible duplicate flush)", stored)
	}

	t.Logf("✅ Concurrent shutdown test passed: %d goroutines, no duplicates", numGoroutines)
}

// Test race conditions with -race flag
func TestMemoryService_RaceDetection(t *testing.T) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    10,
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	// Mix of operations that could race
	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				messages := []*pb.Message{
					{
						Role: pb.Role_ROLE_USER,
						Content: []*pb.Part{
							{Part: &pb.Part_Text{Text: "test"}},
						},
					},
				}
				_ = memoryService.AddBatchToHistory("test-session", messages)
			}
		}(i)
	}

	// Reader goroutine (GetRecentHistory)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = memoryService.GetRecentHistory("test-session")
			time.Sleep(time.Microsecond * 10)
		}
	}()

	// Clear goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 50)
		_ = memoryService.ClearHistory("other-session")
	}()

	wg.Wait()

	// Final shutdown
	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	t.Logf("✅ Race detection test passed (run with -race flag to verify)")
}

// Benchmark concurrent performance
func BenchmarkMemoryService_ConcurrentWrites(b *testing.B) {
	sessionService := newMockSessionService()
	workingMemory := &mockWorkingMemory{}
	longTermMemory := newMockLongTermMemory()

	config := LongTermConfig{
		Enabled:      true,
		BatchSize:    10,
		StorageScope: StorageScopeAll,
	}

	memoryService := NewMemoryService(
		"test-agent",
		sessionService,
		workingMemory,
		longTermMemory,
		config,
	)

	message := []*pb.Message{
		{
			Role: pb.Role_ROLE_USER,
			Content: []*pb.Part{
				{Part: &pb.Part_Text{Text: "test message"}},
			},
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = memoryService.AddBatchToHistory("test-session", message)
		}
	})

	_ = memoryService.Shutdown()
}
