package reasoning

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/tools"
)

type ReasoningResponse struct {
	Answer      string                      `json:"answer"`
	Context     []databases.SearchResult    `json:"context,omitempty"`
	Sources     []string                    `json:"sources,omitempty"`
	ToolResults map[string]tools.ToolResult `json:"tool_results,omitempty"`
	TokensUsed  int                         `json:"tokens_used"`
	Duration    time.Duration               `json:"duration"`
	Confidence  float64                     `json:"confidence,omitempty"`
}

type LLMService interface {
	Generate(messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, error)

	GenerateStreaming(messages []*pb.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]*protocol.ToolCall, int, error)

	GenerateStructured(messages []*pb.Message, tools []llms.ToolDefinition, config *llms.StructuredOutputConfig) (string, []*protocol.ToolCall, int, error)

	SupportsStructuredOutput() bool
}

type ToolService interface {
	ExecuteToolCall(ctx context.Context, toolCall *protocol.ToolCall) (string, map[string]interface{}, error)

	GetAvailableTools() []llms.ToolDefinition

	GetTool(name string) (interface{}, error)
}

type ContextService interface {
	SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error)
	ExtractSources(context []databases.SearchResult) []string
}

type PromptService interface {
	BuildMessages(ctx context.Context, query string, slots PromptSlots, currentToolConversation []*pb.Message, additionalContext string) ([]*pb.Message, error)
}

type SessionService interface {
	AppendMessage(sessionID string, message *pb.Message) error

	AppendMessages(sessionID string, messages []*pb.Message) error

	GetMessages(sessionID string, limit int) ([]*pb.Message, error)
	GetMessagesWithOptions(sessionID string, opts LoadOptions) ([]*pb.Message, error)
	GetMessageCount(sessionID string) (int, error)

	GetOrCreateSessionMetadata(sessionID string) (*SessionMetadata, error)
	DeleteSession(sessionID string) error
	SessionCount() int
}

type SessionMetadata struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]interface{}
}

type LoadOptions struct {
	Limit int

	FromMessageID string

	Roles []pb.Role
}

type HistoryService interface {
	GetRecentHistory(sessionID string) ([]*pb.Message, error)
	AddToHistory(sessionID string, msg *pb.Message) error
	AddBatchToHistory(sessionID string, messages []*pb.Message) error
	ClearHistory(sessionID string) error
}

type StatusNotifiable interface {
	SetStatusNotifier(notifier func(message string))
}

type AgentRegistryEntry struct {
	ID         string
	Card       *pb.AgentCard
	Visibility string
}

type AgentRegistryService interface {
	ListAgents() []AgentRegistryEntry

	GetAgent(id string) (AgentRegistryEntry, bool)

	FilterAgents(ids []string) []AgentRegistryEntry
}

type TaskService interface {
	CreateTask(ctx context.Context, contextID string, initialMessage *pb.Message) (*pb.Task, error)
	GetTask(ctx context.Context, taskID string) (*pb.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error
	AddTaskArtifact(ctx context.Context, taskID string, artifact *pb.Artifact) error
	AddTaskMessage(ctx context.Context, taskID string, message *pb.Message) error
	CancelTask(ctx context.Context, taskID string) (*pb.Task, error)
	ListTasks(ctx context.Context, contextID string, status pb.TaskState, pageSize int32, pageToken string) ([]*pb.Task, string, int32, error)
	ListTasksByContext(ctx context.Context, contextID string) ([]*pb.Task, error)
	SubscribeToTask(ctx context.Context, taskID string) (<-chan *pb.StreamResponse, error)
	Close() error
}

type AgentServices interface {
	GetConfig() config.ReasoningConfig

	LLM() LLMService
	Tools() ToolService
	Context() ContextService
	Prompt() PromptService
	Session() SessionService
	History() HistoryService
	Registry() AgentRegistryService
	Task() TaskService
}

type DefaultAgentServices struct {
	config          config.ReasoningConfig
	llmService      LLMService
	toolService     ToolService
	contextService  ContextService
	promptService   PromptService
	sessionService  SessionService
	historyService  HistoryService
	registryService AgentRegistryService
	taskService     TaskService
}

func NewAgentServices(
	config config.ReasoningConfig,
	llmService LLMService,
	toolService ToolService,
	contextService ContextService,
	promptService PromptService,
	sessionService SessionService,
	historyService HistoryService,
	registryService AgentRegistryService,
	taskService TaskService,
) AgentServices {
	return &DefaultAgentServices{
		config:          config,
		llmService:      llmService,
		toolService:     toolService,
		contextService:  contextService,
		promptService:   promptService,
		sessionService:  sessionService,
		historyService:  historyService,
		registryService: registryService,
		taskService:     taskService,
	}
}

func (s *DefaultAgentServices) GetConfig() config.ReasoningConfig {
	return s.config
}

func (s *DefaultAgentServices) LLM() LLMService {
	return s.llmService
}

func (s *DefaultAgentServices) Tools() ToolService {
	return s.toolService
}

func (s *DefaultAgentServices) Context() ContextService {
	return s.contextService
}

func (s *DefaultAgentServices) Prompt() PromptService {
	return s.promptService
}

func (s *DefaultAgentServices) Session() SessionService {
	return s.sessionService
}

func (s *DefaultAgentServices) History() HistoryService {
	return s.historyService
}

func (s *DefaultAgentServices) Registry() AgentRegistryService {
	return s.registryService
}

func (s *DefaultAgentServices) Task() TaskService {
	return s.taskService
}
