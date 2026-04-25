package claudesdk

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// --- RED: Tests for ClaudeSDKClient ---

func TestNewClient_DefaultOptions(t *testing.T) {
	c := NewClient(nil)
	if c == nil {
		t.Fatal("NewClient(nil) returned nil")
	}
	if c.opts == nil {
		t.Fatal("opts should not be nil when nil is passed")
	}
	if c.connected {
		t.Fatal("new client should not be connected")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	opts := &ClaudeAgentOptions{Model: "test-model"}
	c := NewClient(opts)
	if c.opts.Model != "test-model" {
		t.Fatalf("expected model %q, got %q", "test-model", c.opts.Model)
	}
}

func TestClient_SendQuery_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.SendQuery(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error when sending query on unconnected client")
	}
}

func TestClient_Interrupt_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.Interrupt(context.Background())
	if err == nil {
		t.Fatal("expected error when interrupting unconnected client")
	}
}

func TestClient_SetPermissionMode_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.SetPermissionMode(context.Background(), PermissionModeDontAsk)
	if err == nil {
		t.Fatal("expected error when setting permission mode on unconnected client")
	}
}

func TestClient_SetModel_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.SetModel(context.Background(), "claude-sonnet")
	if err == nil {
		t.Fatal("expected error when setting model on unconnected client")
	}
}

func TestClient_GetMCPStatus_NotConnected(t *testing.T) {
	c := NewClient(nil)
	_, err := c.GetMCPStatus(context.Background())
	if err == nil {
		t.Fatal("expected error when getting MCP status on unconnected client")
	}
}

func TestClient_GetContextUsage_NotConnected(t *testing.T) {
	c := NewClient(nil)
	_, err := c.GetContextUsage(context.Background())
	if err == nil {
		t.Fatal("expected error when getting context usage on unconnected client")
	}
}

func TestClient_RewindFiles_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.RewindFiles(context.Background(), "msg-1")
	if err == nil {
		t.Fatal("expected error when rewinding on unconnected client")
	}
}

func TestClient_ReconnectMcpServer_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.ReconnectMcpServer(context.Background(), "server1")
	if err == nil {
		t.Fatal("expected error on unconnected client")
	}
}

func TestClient_ToggleMcpServer_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.ToggleMcpServer(context.Background(), "server1", true)
	if err == nil {
		t.Fatal("expected error on unconnected client")
	}
}

func TestClient_StopTask_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.StopTask(context.Background(), "task-1")
	if err == nil {
		t.Fatal("expected error on unconnected client")
	}
}

func TestClient_GetServerInfo_NotConnected(t *testing.T) {
	c := NewClient(nil)
	info := c.GetServerInfo()
	if info != nil {
		t.Fatal("expected nil server info on unconnected client")
	}
}

func TestClient_GetInitResult_NotConnected(t *testing.T) {
	c := NewClient(nil)
	info := c.GetInitResult()
	if info != nil {
		t.Fatal("expected nil init result on unconnected client")
	}
}

func TestClient_Close_NotConnected(t *testing.T) {
	c := NewClient(nil)
	err := c.Close()
	if err != nil {
		t.Fatalf("expected no error closing unconnected client, got: %v", err)
	}
}

// --- Tests with MockTransport (connected client) ---

// setupConnectedClient creates a ClaudeSDKClient backed by a MockTransport.
func setupConnectedClient(t *testing.T, dataMessages ...json.RawMessage) (*ClaudeSDKClient, *MockTransport) {
	t.Helper()

	mt := NewMockTransport(dataMessages...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client := NewClient(&ClaudeAgentOptions{})
	client.transport = mt
	if err := mt.Connect(ctx); err != nil {
		t.Fatalf("mock connect: %v", err)
	}
	client.handler = newQueryHandler(mt, client.opts)
	client.msgCh = client.handler.runMessageRouter(ctx)
	if _, err := client.handler.initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	client.connected = true
	return client, mt
}

func TestClient_ConnectedFlow_SendAndReceive(t *testing.T) {
	msgs := QuickMockMessages("hello world", 0.01)
	client, _ := setupConnectedClient(t, msgs...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.SendQuery(ctx, "say hello"); err != nil {
		t.Fatalf("SendQuery: %v", err)
	}

	var gotAssistant, gotResult bool
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case AssistantMessage:
			text := GetTextContent(m)
			if text != "hello world" {
				t.Fatalf("expected 'hello world', got %q", text)
			}
			gotAssistant = true
		case ResultMessage:
			if m.CostUSD != 0.01 {
				t.Fatalf("expected cost 0.01, got %f", m.CostUSD)
			}
			gotResult = true
		}
	}
	if !gotAssistant {
		t.Fatal("did not receive assistant message")
	}
	if !gotResult {
		t.Fatal("did not receive result message")
	}
}

func TestClient_GetServerInfo_Connected(t *testing.T) {
	client, _ := setupConnectedClient(t)

	info := client.GetServerInfo()
	if info == nil {
		t.Fatal("expected non-nil server info after connect")
	}
	if info.Account.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %q", info.Account.Email)
	}
}

func TestClient_GetInitResult_Connected(t *testing.T) {
	client, _ := setupConnectedClient(t)

	result := client.GetInitResult()
	if result == nil {
		t.Fatal("expected non-nil init result")
	}
	if len(result.Models) == 0 {
		t.Fatal("expected at least one model")
	}
}

func TestClient_Close_Connected(t *testing.T) {
	client, mt := setupConnectedClient(t)

	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if client.connected {
		t.Fatal("client should be disconnected after Close")
	}
	mt.mu.Lock()
	closed := mt.closed
	mt.mu.Unlock()
	if !closed {
		t.Fatal("transport should be closed")
	}
}

func TestClient_ReceiveMessages_ReturnsChannel(t *testing.T) {
	client, _ := setupConnectedClient(t)

	ctx := context.Background()
	ch := client.ReceiveMessages(ctx)
	if ch == nil {
		t.Fatal("expected non-nil messages channel")
	}
}
