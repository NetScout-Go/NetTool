// Package plugins provides WebSocket streaming for plugin execution.
package plugins

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
	"github.com/gorilla/websocket"
)

// StreamMessage represents a WebSocket message for plugin execution.
type StreamMessage struct {
	Type        string      `json:"type"`
	ExecutionID string      `json:"executionId,omitempty"`
	PluginID    string      `json:"pluginId,omitempty"`
	Data        interface{} `json:"data,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// MessageType constants for WebSocket messages.
const (
	// Execution lifecycle messages
	MsgTypeExecutionStarted   = "execution_started"
	MsgTypeExecutionProgress  = "execution_progress"
	MsgTypeExecutionCompleted = "execution_completed"
	MsgTypeExecutionFailed    = "execution_failed"
	MsgTypeExecutionCancelled = "execution_cancelled"

	// Request messages from client
	MsgTypeCancelExecution    = "cancel_execution"
	MsgTypeGetExecutionStatus = "get_execution_status"
	MsgTypeSubscribe          = "subscribe"
	MsgTypeUnsubscribe        = "unsubscribe"
)

// ExecutionStreamClient represents a WebSocket client for plugin streaming.
type ExecutionStreamClient struct {
	conn          *websocket.Conn
	subscriptions map[string]bool // executionId -> subscribed
	allExecutions bool            // subscribe to all executions
	mu            sync.Mutex
}

// NewExecutionStreamClient creates a new streaming client.
func NewExecutionStreamClient(conn *websocket.Conn) *ExecutionStreamClient {
	return &ExecutionStreamClient{
		conn:          conn,
		subscriptions: make(map[string]bool),
		allExecutions: false,
	}
}

// Subscribe subscribes to an execution.
func (c *ExecutionStreamClient) Subscribe(executionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[executionID] = true
}

// SubscribeAll subscribes to all executions.
func (c *ExecutionStreamClient) SubscribeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allExecutions = true
}

// Unsubscribe unsubscribes from an execution.
func (c *ExecutionStreamClient) Unsubscribe(executionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscriptions, executionID)
}

// IsSubscribed checks if client is subscribed to an execution.
func (c *ExecutionStreamClient) IsSubscribed(executionID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.allExecutions || c.subscriptions[executionID]
}

// Send sends a message to the client.
func (c *ExecutionStreamClient) Send(msg StreamMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(msg)
}

// ExecutionStreamManager manages WebSocket clients for execution streaming.
type ExecutionStreamManager struct {
	clients  map[*ExecutionStreamClient]bool
	executor *PluginExecutor
	mu       sync.RWMutex
}

// NewExecutionStreamManager creates a new stream manager.
func NewExecutionStreamManager(executor *PluginExecutor) *ExecutionStreamManager {
	return &ExecutionStreamManager{
		clients:  make(map[*ExecutionStreamClient]bool),
		executor: executor,
	}
}

// RegisterClient registers a new WebSocket client.
func (m *ExecutionStreamManager) RegisterClient(conn *websocket.Conn) *ExecutionStreamClient {
	client := NewExecutionStreamClient(conn)

	m.mu.Lock()
	m.clients[client] = true
	m.mu.Unlock()

	return client
}

// UnregisterClient removes a WebSocket client.
func (m *ExecutionStreamManager) UnregisterClient(client *ExecutionStreamClient) {
	m.mu.Lock()
	delete(m.clients, client)
	m.mu.Unlock()
}

// Broadcast sends a message to all subscribed clients.
func (m *ExecutionStreamManager) Broadcast(executionID string, msg StreamMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if client.IsSubscribed(executionID) {
			go func(c *ExecutionStreamClient) {
				if err := c.Send(msg); err != nil {
					log.Printf("Error sending message to client: %v", err)
				}
			}(client)
		}
	}
}

// HandleClientMessage processes incoming messages from a client.
func (m *ExecutionStreamManager) HandleClientMessage(client *ExecutionStreamClient, rawMsg []byte) {
	var msg StreamMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		log.Printf("Error parsing client message: %v", err)
		return
	}

	switch msg.Type {
	case MsgTypeSubscribe:
		if msg.ExecutionID != "" {
			client.Subscribe(msg.ExecutionID)
		} else {
			client.SubscribeAll()
		}

	case MsgTypeUnsubscribe:
		if msg.ExecutionID != "" {
			client.Unsubscribe(msg.ExecutionID)
		}

	case MsgTypeCancelExecution:
		if msg.ExecutionID != "" {
			if err := m.executor.CancelExecution(msg.ExecutionID); err != nil {
				errMsg := StreamMessage{
					Type:        "error",
					ExecutionID: msg.ExecutionID,
					Error:       err.Error(),
				}
				if sendErr := client.Send(errMsg); sendErr != nil {
					log.Printf("Error sending error message: %v", sendErr)
				}
			}
		}

	case MsgTypeGetExecutionStatus:
		if msg.ExecutionID != "" {
			if exec := m.executor.GetExecution(msg.ExecutionID); exec != nil {
				result := exec.GetResult()
				statusMsg := StreamMessage{
					Type:        "execution_status",
					ExecutionID: msg.ExecutionID,
					PluginID:    result.PluginID,
					Data:        result,
				}
				if err := client.Send(statusMsg); err != nil {
					log.Printf("Error sending status message: %v", err)
				}
			}
		}
	}
}

// CreateProgressListener creates a progress listener for an execution.
func (m *ExecutionStreamManager) CreateProgressListener(executionID string) ProgressListener {
	return func(result *ExecutionResult) {
		var msgType string
		switch result.Status {
		case StatusRunning:
			if result.Progress != nil {
				msgType = MsgTypeExecutionProgress
			} else {
				msgType = MsgTypeExecutionStarted
			}
		case StatusCompleted:
			msgType = MsgTypeExecutionCompleted
		case StatusFailed:
			msgType = MsgTypeExecutionFailed
		case StatusCancelled:
			msgType = MsgTypeExecutionCancelled
		default:
			return
		}

		msg := StreamMessage{
			Type:        msgType,
			ExecutionID: executionID,
			PluginID:    result.PluginID,
			Data:        result,
		}

		m.Broadcast(executionID, msg)
	}
}

// ExecuteWithStreaming executes a plugin with streaming progress updates.
func (m *ExecutionStreamManager) ExecuteWithStreaming(pluginID string, params map[string]interface{}, config *types.PluginExecutionConfig) (*ExecutionContext, error) {
	// Create execution context
	execCtx := m.executor.Execute(pluginID, params, config)

	// Add progress listener
	execCtx.AddProgressListener(m.CreateProgressListener(execCtx.ID))

	return execCtx, nil
}
