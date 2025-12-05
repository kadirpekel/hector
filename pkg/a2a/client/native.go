package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/a2aproject/a2a-go/a2aclient/agentcard"

	"github.com/kadirpekel/hector/pkg/httpclient"
)

// NativeClient wraps a2aclient.Client to implement our A2AClient interface
type NativeClient struct {
	client   *a2aclient.Client
	agentID  string
	agentCard *a2a.AgentCard
}

// NewNativeClient creates a new native a2a-go client
func NewNativeClient(ctx context.Context, url, agentID, token string, tlsConfig *httpclient.TLSConfig) (*NativeClient, error) {
	// Create HTTP client with TLS configuration if needed
	var httpClient *http.Client
	if tlsConfig != nil && (tlsConfig.InsecureSkipVerify || tlsConfig.CACertificate != "") {
		transport, err := httpclient.ConfigureTLS(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		httpClient = &http.Client{
			Transport: transport,
		}
	}

	// Resolve agent card
	var resolver *agentcard.Resolver
	if httpClient != nil {
		resolver = agentcard.NewResolver(httpClient)
	} else {
		resolver = agentcard.DefaultResolver
	}

	card, err := resolver.Resolve(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent card: %w", err)
	}

	// Create a2aclient
	// Note: Auth token is handled via HTTP headers in the resolver if provided
	// For now, create client without explicit auth options
	// TODO: Add proper auth support via a2aclient factory options
	client, err := a2aclient.NewFromCard(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("failed to create a2a client: %w", err)
	}

	return &NativeClient{
		client:    client,
		agentID:   agentID,
		agentCard: card,
	}, nil
}

func (c *NativeClient) SendMessage(ctx context.Context, message *a2a.Message) (*a2a.Task, error) {
	params := &a2a.MessageSendParams{
		Message: message,
	}
	result, err := c.client.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}
	// SendMessageResult is an Event interface - extract task info
	// The result should contain task information via TaskInfo() method
	taskInfo := result.TaskInfo()
	if taskInfo.TaskID != "" {
		return c.GetTask(ctx, string(taskInfo.TaskID))
	}
	// If we can't extract task, return error
	return nil, fmt.Errorf("unable to extract task from SendMessage result")
}

func (c *NativeClient) StreamMessage(ctx context.Context, message *a2a.Message) (<-chan *a2a.Event, error) {
	params := &a2a.MessageSendParams{
		Message: message,
	}
	eventStream := c.client.SendStreamingMessage(ctx, params)

	ch := make(chan *a2a.Event, 10)
	go func() {
		defer close(ch)
		for event, err := range eventStream {
			if err != nil {
				return
			}
			// event is a2a.Event (interface), store it in a variable and send pointer
			if event != nil {
				evt := event
				ch <- &evt
			}
		}
	}()

	return ch, nil
}

func (c *NativeClient) GetAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	if c.agentCard != nil {
		return c.agentCard, nil
	}
	return c.client.GetAgentCard(ctx)
}

func (c *NativeClient) GetTask(ctx context.Context, taskID string) (*a2a.Task, error) {
	params := &a2a.TaskQueryParams{
		ID: a2a.TaskID(taskID),
	}
	return c.client.GetTask(ctx, params)
}

func (c *NativeClient) ListTasks(ctx context.Context, contextID string, status a2a.TaskState, pageSize int32, pageToken string) ([]*a2a.Task, string, int32, error) {
	// TODO: Implement ListTasks using a2a-go client
	// For now, return empty result
	return nil, "", 0, fmt.Errorf("ListTasks not yet implemented in native client")
}

func (c *NativeClient) CancelTask(ctx context.Context, taskID string) (*a2a.Task, error) {
	params := &a2a.TaskIDParams{
		ID: a2a.TaskID(taskID),
	}
	return c.client.CancelTask(ctx, params)
}

func (c *NativeClient) GetAgentID() string {
	return c.agentID
}

func (c *NativeClient) Close() error {
	return c.client.Destroy()
}

