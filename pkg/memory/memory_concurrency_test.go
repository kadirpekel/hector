package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

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

func (m *mockSessionService) UpdateSessionMetadata(sessionID string, metadata map[string]interface{}) error {
	// Mock implementation - just return success
	return nil
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
	storeCh chan bool
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
						Parts: []*pb.Part{
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

func TestMemoryService_ConcurrentLongTermBatching(t *testing.T) {
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
						Parts: []*pb.Part{
							{Part: &pb.Part_Text{Text: "test message"}},
						},
					},
				}

				_ = memoryService.AddBatchToHistory(sessionID, messages)

				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	stored := longTermMemory.GetStoredCount("test-agent", "test-session")
	expected := numGoroutines * messagesPerGoroutine

	if stored != expected {
		t.Errorf("Expected %d stored messages, got %d", expected, stored)
	}

	t.Logf("✅ Concurrent long-term batching passed: %d messages stored", stored)
}

func TestMemoryService_ShutdownWithPendingBatches(t *testing.T) {
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

	sessions := []string{"session1", "session2", "session3"}
	messagesPerSession := 50

	for _, sessionID := range sessions {
		for i := 0; i < messagesPerSession; i++ {
			messages := []*pb.Message{
				{
					Role: pb.Role_ROLE_USER,
					Parts: []*pb.Part{
						{Part: &pb.Part_Text{Text: "test message"}},
					},
				},
			}
			_ = memoryService.AddBatchToHistory(sessionID, messages)
		}
	}

	for _, sessionID := range sessions {
		stored := longTermMemory.GetStoredCount("test-agent", sessionID)
		if stored != 0 {
			t.Errorf("Expected 0 stored messages before shutdown for %s, got %d", sessionID, stored)
		}
	}

	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

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

	for i := 0; i < 50; i++ {
		messages := []*pb.Message{
			{
				Role: pb.Role_ROLE_USER,
				Parts: []*pb.Part{
					{Part: &pb.Part_Text{Text: "test message"}},
				},
			},
		}
		_ = memoryService.AddBatchToHistory("test-session", messages)
	}

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

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent shutdown returned error: %v", err)
		}
	}

	stored := longTermMemory.GetStoredCount("test-agent", "test-session")
	if stored != 50 {
		t.Errorf("Expected 50 stored messages, got %d (possible duplicate flush)", stored)
	}

	t.Logf("✅ Concurrent shutdown test passed: %d goroutines, no duplicates", numGoroutines)
}

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

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				messages := []*pb.Message{
					{
						Role: pb.Role_ROLE_USER,
						Parts: []*pb.Part{
							{Part: &pb.Part_Text{Text: "test"}},
						},
					},
				}
				_ = memoryService.AddBatchToHistory("test-session", messages)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = memoryService.GetRecentHistory("test-session")
			time.Sleep(time.Microsecond * 10)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 50)
		_ = memoryService.ClearHistory("other-session")
	}()

	wg.Wait()

	if err := memoryService.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	t.Logf("✅ Race detection test passed (run with -race flag to verify)")
}

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
			Parts: []*pb.Part{
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
